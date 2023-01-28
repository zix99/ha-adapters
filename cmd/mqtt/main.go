package main

import (
	"errors"
	"ha-adapters/cmd/internal/xcli/clilog"
	"ha-adapters/cmd/internal/xcli/climqtt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func runPublisher(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("invalid arg count")
	}

	mqtt, err := climqtt.BuildClientFromFlags(c)
	if err != nil {
		logrus.Fatal(err)
	}

	var (
		topic   = c.Args().Get(0)
		payload = c.Args().Get(1)
	)

	if err := mqtt.PublishString(topic, payload); err != nil {
		return err
	}

	return nil
}

func runListener(c *cli.Context) error {
	mqtt, err := climqtt.BuildClientFromFlags(c)
	if err != nil {
		logrus.Fatal(err)
	}
	defer mqtt.Close()

	topic := "#"
	if c.NArg() > 0 {
		topic = c.Args().First()
	}

	events, err := mqtt.Subscribe(topic)
	if err != nil {
		logrus.Fatal(err)
	}
	for e := range events {
		logrus.Infof("%s: %s", e.Topic(), string(e.Payload()))
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Commands = []*cli.Command{
		{
			Name:      "listen",
			Action:    runListener,
			Flags:     climqtt.Flags,
			UsageText: "<topic|#>",
		},
		{
			Name:   "publish",
			Action: runPublisher,
			Flags:  climqtt.Flags,
		},
	}

	clilog.AdaptForLogSettings(app)

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
