package main

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"github.com/lucagrulla/cw/cloudwatch"
)

var (
	timeFormat = "2006-01-02T15:04:05"
	startTime  = time.Now().UTC().Add(-30 * time.Second).Format(timeFormat)
	version    = "3.0.1"

	cli struct {
		Version    kong.VersionFlag `help:"Show version."`
		Debug      bool             `flag hidden:"" help:"Enable debug mode." short:"d"` //TODO hidden is not working
		awsProfile string           `flag name:"profile" help:"The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file." short:"p" placeholder:"PROFILE"`
		awsRegion  string           `flag name:"region" help:"The target AWS region. By default cw will use the default profile defined in the .aws/credentials file." short:"r" placeholder:"REGION"`
		NoColor    bool             `flag help:"Disable coloured output." short:"c"`

		Tail TailCmd `cmd help:"Tail log groups/streams."`
		Ls   LsCmd   `cmd help:"Show an entity."`
	}
)

func main() {
	// kp.Version(version).Author("Luca Grulla")
	log := log.New(ioutil.Discard, "", log.LstdFlags)

	cwClient := cloudwatch.New(&cli.awsProfile, &cli.awsRegion, log)
	ctx := kong.Parse(&cli, kong.Description("The best way to tail AWS Cloudwatch Logs from your terminal."),
		kong.Vars{
			"startTime": startTime,
			"version":   version,
		}, kong.Bind(cwClient, log))

	if cli.Debug {
		log.SetOutput(os.Stdout)
		log.Println("Debug mode is on.")
	}
	log.Printf("awsProfile: %s, awsRegion: %s\n", cli.awsProfile, cli.awsRegion)

	if cli.NoColor {
		color.NoColor = true
	}

	defer newVersionMsg(version, fetchLatestVersion(), cli.NoColor)
	go versionCheckOnSigterm()

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
