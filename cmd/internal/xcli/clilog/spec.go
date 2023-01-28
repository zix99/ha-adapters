package clilog

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func AdaptForLogSettings(app *cli.App) {
	app.Flags = append(app.Flags, []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Enables debug messages",
		},
		&cli.BoolFlag{
			Name:  "trace",
			Usage: "Enables trace messages",
		},
	}...)

	oldBefore := app.Before
	app.Before = func(ctx *cli.Context) error {
		if ctx.Bool("verbose") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if ctx.Bool("trace") {
			logrus.SetLevel(logrus.TraceLevel)
		}

		if oldBefore != nil {
			return oldBefore(ctx)
		}
		return nil
	}
}
