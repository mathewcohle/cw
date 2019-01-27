package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/fatih/color"
)

func timestampToTime(timeStamp string, local bool) (time.Time, error) {
	var zone *time.Location
	if local {
		zone = time.Local
	} else {
		zone = time.UTC
	}
	if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02", timeStamp, zone)
		return t, nil
	} else if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}$`).MatchString(timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15", timeStamp, zone)
		return t, nil
	} else if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$`).MatchString(timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15:04", timeStamp, zone)
		return t, nil
	} else if regexp.MustCompile(`^\d{1,2}$`).MatchString(timeStamp) {
		y, m, d := time.Now().In(zone).Date()
		t, _ := strconv.Atoi(timeStamp)
		return time.Date(y, m, d, t, 0, 0, 0, zone), nil
	} else if res := regexp.MustCompile(`^(?P<Hour>\d{1,2}):(?P<Minute>\d{2})$`).FindStringSubmatch(timeStamp); res != nil {
		y, m, d := time.Now().Date()

		t, _ := strconv.Atoi(res[1])
		mm, _ := strconv.Atoi(res[2])

		return time.Date(y, m, d, t, mm, 0, 0, zone), nil
	} else if regexp.MustCompile(`^\d{1,}h$|^\d{1,}m$|^\d{1,}h\d{1,}m$`).MatchString(timeStamp) {
		d, _ := time.ParseDuration(timeStamp)

		t := time.Now().In(zone).Add(-d)
		y, m, dd := t.Date()
		return time.Date(y, m, dd, t.Hour(), t.Minute(), 0, 0, zone), nil
	}

	//TODO check even last scenario and if it's not a recognized pattern throw an error
	t, err := time.ParseInLocation("2006-01-02T15:04:05", timeStamp, zone)
	if err != nil {
		return t, err
	}
	return t, nil
}

func formatLogMsg(ev logEvent, tail tailCmd) string {
	msg := *ev.logEvent.Message
	if tail.PrintEventID {
		msg = fmt.Sprintf("%s - %s", color.YellowString(*ev.logEvent.EventId), msg)
	}
	if tail.PrintStreamName {
		msg = fmt.Sprintf("%s - %s", color.BlueString(*ev.logEvent.LogStreamName), msg)
	}

	if tail.PrintGroupName {
		msg = fmt.Sprintf("%s - %s", color.CyanString(ev.logGroup), msg)
	}

	if tail.PrintTimeStamp {
		eventTimestamp := *ev.logEvent.Timestamp / 1000
		ts := time.Unix(eventTimestamp, 0).Format(timeFormat)
		msg = fmt.Sprintf("%s - %s", color.GreenString(ts), msg)
	}
	return msg
}
