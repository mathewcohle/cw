package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/lucagrulla/cw/cloudwatch"
)

type logEvent struct {
	logEvent cloudwatchlogs.FilteredLogEvent
	logGroup string
}

type LsCmd struct {
	GroupsCmd  LsGroupsCmd  `cmd name:"groups" help:"Show all groups."`
	StreamsCmd LsStreamsCmd `cmd name:"streams" help:"Show all streams in a given log group."`
}

type LsGroupsCmd struct {
}

func (ls *LsGroupsCmd) Run(c *cloudwatch.CW) error {
	for msg := range c.LsGroups() {
		fmt.Println(*msg)
	}
	return nil
}

type LsStreamsCmd struct {
	LogGroupName string `arg required help:"The log group name."`
}

func (ls LsStreamsCmd) Run(c *cloudwatch.CW) error {
	for msg := range c.LsStreams(&ls.LogGroupName, nil) {
		fmt.Println(*msg)
	}
	return nil
}

type TailCmd struct {
	LogGroupStreamName []string `arg required name:"groupName[:logStreamPrefix]" help:"The log group and stream name, with group:prefix syntax. Stream name can be just the prefix. If no stream name is specified all stream names in the given group will be tailed. Multiple group/stream tuple can be passed. e.g. cw tail group1:prefix1 group2:prefix2 group3:prefix3."`
	Follow             bool     `"flag help:Don't stop when the end of streams is reached, but rather wait for additional data to be appended." short:"f" default:"false"`
	PrintTimeStamp     bool     `flag name:"timestamp" help:"Print the event timestamp." short:"t" default:"false"`
	PrintStreamName    bool     `flag name:"stream-name" help:"Print the log stream name this event belongs to." short:"s"`
	PrintGroupName     bool     `flag name:"group-name" help:"Print the log group name this event belongs to." short:"n"`
	PrintEventID       bool     `flag name:"event-id" help:"Print the event Id." short:"i" default:"false"`
	StartTime          string   `flag short:"b" default:"NOW" help:"The UTC start time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of hours and minutes prior to the present. Denote hours with 'h' and minutes with 'm' i.e. 80m, 4h30m. If just time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format: 2017-02-27[T09[:00[:00]]."`
	EndTime            string   `flag short:"e" default:"" help:"The UTC end time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of hours and minutes prior to the present. Denote hours with 'h' and minutes with 'm' i.e. 80m, 4h30m. If just time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format: 2017-02-27[T09[:00[:00]]."`
	local              bool     `flag help:"Treat date and time in Local timezone." short:"l" default:"false"`
	grep               string   `flag help:"Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax." short:"g" default:""`
	grepv              string   `flag help:"Equivalent of grep --invert-match. Invert match pattern to filter logs by." short:"v" default:""`
}

func (tail *TailCmd) BeforeApply() error {
	fmt.Println("before apply", tail)
	if tail.StartTime == "NOW" {
		tail.StartTime = startTime
	}
	return nil
}
func (tail *TailCmd) BeforeResolve() error {
	fmt.Println("before resolve")
	info, _ := os.Stdin.Stat()
	if info.Size() > 0 {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := scanner.Text()
		// reader := bufio.NewReader(os.Stdin)
		// input, _ := reader.ReadString('\n')
		tail.LogGroupStreamName = append(tail.LogGroupStreamName, input)
		fmt.Println(input, tail.LogGroupStreamName)
	}
	return nil
}

func (tail TailCmd) Run(cwl *cloudwatch.CW, logger *log.Logger) error {
	st, err := timestampToTime(tail.StartTime, tail.local)
	if err != nil {
		log.Fatalf("can't parse %s as a valid start date/time", tail.StartTime)
	}
	var et time.Time
	if tail.EndTime != "" {
		endT, errr := timestampToTime(tail.EndTime, tail.local)
		if errr != nil {
			log.Fatalf("can't parse %s as a valid end date/time", tail.EndTime)
		} else {
			et = endT
		}
	}
	out := make(chan *logEvent)

	var wg sync.WaitGroup

	triggerChannels := make([]chan<- time.Time, len(tail.LogGroupStreamName))

	coordinator := &tailCoordinator{log: logger}
	for idx, gs := range tail.LogGroupStreamName {
		trigger := make(chan time.Time, 1)
		go func(groupStream string) {
			tokens := strings.Split(groupStream, ":")
			var prefix string
			group := tokens[0]
			if len(tokens) > 1 && tokens[1] != "*" {
				prefix = tokens[1]
			}
			for c := range cwl.Tail(&group, &prefix, &tail.Follow, &st, &et, &tail.grep, &tail.grepv, trigger) {
				out <- &logEvent{logEvent: *c, logGroup: group}
			}
			coordinator.remove(trigger)
			wg.Done()
		}(gs)
		triggerChannels[idx] = trigger
		wg.Add(1)
	}

	coordinator.start(triggerChannels)

	go func() {
		wg.Wait()
		logger.Println("closing main channel...")
		close(out)
	}()

	for logEv := range out {
		fmt.Println(formatLogMsg(*logEv, tail))
	}
	return nil
}
