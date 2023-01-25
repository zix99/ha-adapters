package main

import (
	"ha-adapters/pkg/amcrest"
	"ha-adapters/pkg/comms"
	"ha-adapters/pkg/comms/homeassistant"
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v2"
)

// https://github.com/dchesterton/amcrest2mqtt/blob/9917b41381c62ef32281c0f508caf3254cf76968/src/amcrest2mqtt.py

func runAD410(c *cli.Context) error {
	var (
		amcrestUrl      = c.String("ad410-url")
		amcrestUsername = c.String("ad410-username")
		amcrestPassword = c.String("ad410-password")
		mqttUri         = c.String("mqtt-uri")
		mqttUsername    = c.String("mqtt-username")
		mqttPassword    = c.String("mqtt-password")
	)

	if amcrestUrl == "" || mqttUri == "" {
		logrus.Fatal("Missing config")
	}

	// setup and connect to doorbell
	doorbell, err := amcrest.ConnectAmcrest(amcrestUrl, amcrestUsername, amcrestPassword)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Println("Serial: " + doorbell.SerialNumber)
	logrus.Println("Type  : " + doorbell.DeviceType)
	logrus.Println(doorbell.SoftwareVersion)

	// Setup MQTT and eventing
	mqtt, err := comms.NewMqtt(mqttUri, mqttUsername, mqttPassword)
	if err != nil {
		logrus.Fatal(err)
	}
	defer mqtt.Close()

	ha := homeassistant.NewHomeAssistant(mqtt)
	defer ha.Close()

	device := comms.DeviceClass{
		DeviceName:   "Amcrest " + doorbell.DeviceType,
		Manufacturer: "Amcrest",
		Model:        doorbell.DeviceType,
		Identifier:   "ad410-" + doorbell.SerialNumber,
		Version:      doorbell.SoftwareVersion,
	}

	dButton := comms.Sensor{
		DeviceClass: device,
		Name:        "Doorbell",
		Type:        comms.DT_BINARY_SENSOR,
		Icon:        "mdi:doorbell",
	}

	dHuman := comms.Sensor{
		DeviceClass: device,
		Name:        "Human",
		Type:        comms.DT_BINARY_SENSOR,
		ClassType:   comms.DC_MOTION,
	}

	dMotion := comms.Sensor{
		DeviceClass: device,
		Name:        "Motion",
		Type:        comms.DT_BINARY_SENSOR,
		ClassType:   comms.DC_MOTION,
	}

	dStorageUsedPercent := comms.Sensor{
		DeviceClass:       device,
		Type:              comms.DT_SENSOR,
		Name:              "Storage Used %",
		UnitOfMeasurement: "%",
	}

	ha.Advertise(&dButton)
	ha.Advertise(&dHuman)
	ha.Advertise(&dMotion)
	ha.Advertise(&dStorageUsedPercent)

	// Core event loop
	stream := doorbell.OpenReliableEventStream(10)

	// exit signal
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	defer signal.Stop(sigint)

	metdataTicker := time.NewTicker(5 * time.Minute)
	defer metdataTicker.Stop()

LOOP:
	for {
		select {
		case event, ok := <-stream:
			if !ok || event.Err != nil {
				logrus.Warnf("Stream ended, aborting.")
				break LOOP
			}

			switch event.Code {
			case "VideoMotion":
				mqtt.PublishState(&dMotion, comms.StateStr(event.Action == "Start"))
			case "CrossRegionDetection":
				if gjson.Get(event.Data, "ObjectType").String() == "Human" {
					mqtt.PublishState(&dHuman, comms.StateStr(event.Action == "Start"))
				}
			case "_DoTalkAction_": // TODO
				state := comms.StateStr(gjson.Get(event.Data, "Action").String() == "Invite")
				mqtt.PublishState(&dButton, state)
			}
		case <-sigint:
			logrus.Info("Received interrupt")
			break LOOP

		case <-metdataTicker.C:
			logrus.Info("Updating metadata...")
			// TODO
			logrus.Debug(doorbell.GetStorageInfo())
			mqtt.PublishState(&dStorageUsedPercent, "0")
		}
	}

	// Go down
	logrus.Info("Shutting down...")
	return nil
}

func main() {
	app := cli.NewApp()
	app.Usage = "Amcrest AD410 to MQTT (Home-assistant)"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "ad410-url",
			EnvVars:  []string{"AD410_URL"},
			Usage:    "URL of the AD410 doorbell",
			Required: true,
		},
		&cli.StringFlag{
			Name:        "ad410-username",
			EnvVars:     []string{"AD410_USERNAME"},
			DefaultText: "admin",
			Usage:       "AD410 username",
		},
		&cli.StringFlag{
			Name:    "ad410-password",
			EnvVars: []string{"AD410_PASSWORD"},
			Usage:   "AD410 password",
		},
		&cli.StringFlag{
			Name:     "mqtt-uri",
			EnvVars:  []string{"MQTT_URI"},
			Usage:    "Set MQTT broker",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "mqtt-username",
			EnvVars: []string{"MQTT_USERNAME"},
			Usage:   "MQTT username",
		},
		&cli.StringFlag{
			Name:    "mqtt-password",
			EnvVars: []string{"MQTT_PASSWORD"},
			Usage:   "MQTT password",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Verbose log mode",
		},
		&cli.BoolFlag{
			Name:  "trace",
			Usage: "Even more logging",
		},
	}
	app.Before = func(ctx *cli.Context) error {
		if ctx.Bool("verbose") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if ctx.Bool("trace") {
			logrus.SetLevel(logrus.TraceLevel)
		}
		return nil
	}
	app.Action = runAD410

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
