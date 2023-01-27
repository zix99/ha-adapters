package main

import (
	"fmt"
	"ha-adapters/pkg/amcrest"
	"ha-adapters/pkg/comms"
	"ha-adapters/pkg/comms/homeassistant"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v2"
)

// https://github.com/dchesterton/amcrest2mqtt/blob/9917b41381c62ef32281c0f508caf3254cf76968/src/amcrest2mqtt.py
// Setting config: https://github.com/rroller/dahua/issues/52
// Downloading data: https://github.com/rroller/dahua/issues/97

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
		DeviceName:   "Amcrest(HAA) " + doorbell.DeviceType,
		Manufacturer: "Amcrest(HAA)",
		Model:        doorbell.DeviceType,
		Identifier:   "ad410b-" + doorbell.SerialNumber,
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
		Name:              "Storage Used Percent",
		UnitOfMeasurement: "%",
	}

	dStorageUsed := comms.Sensor{
		DeviceClass:       device,
		Type:              comms.DT_SENSOR,
		Name:              "Storage Used",
		UnitOfMeasurement: "GB",
	}

	ha.Advertise(&dButton)
	mqtt.PublishState(&dButton, comms.STATE_OFF)

	ha.Advertise(&dHuman)
	mqtt.PublishState(&dHuman, comms.STATE_OFF)

	ha.Advertise(&dMotion)
	mqtt.PublishState(&dMotion, comms.STATE_OFF)

	ha.Advertise(&dStorageUsedPercent)
	ha.Advertise(&dStorageUsed)

	// Core event loop
	stream := doorbell.OpenReliableEventStream(10)

	// exit signal
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	defer signal.Stop(sigint)

	metadataTicker := time.NewTicker(5 * time.Minute)
	defer metadataTicker.Stop()

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
				if gjson.Get(event.Data, "Object.ObjectType").String() == "Human" {
					mqtt.PublishState(&dHuman, comms.StateStr(event.Action == "Start"))
				}
			case "_DoTalkAction_":
				state := comms.StateStr(gjson.Get(event.Data, "Action").String() == "Invite")
				mqtt.PublishState(&dButton, state)
			case "NewFile": // TODO
				path := gjson.Get(event.Data, "File").String()
				go doorbell.DownloadFileTo(path, "temp/"+fmt.Sprintf("%d", time.Now().UnixMilli()))
			}
		case <-sigint:
			logrus.Info("Received interrupt")
			break LOOP

		case <-metadataTicker.C:
			logrus.Info("Updating metadata...")
			info, err := doorbell.GetStorageInfo()
			if err == nil {
				logrus.Debug(info)
				totalBytes, err0 := strconv.ParseFloat(info["list.info[0].Detail[0].TotalBytes"], 64)
				usedBytes, err1 := strconv.ParseFloat(info["list.info[0].Detail[0].UsedBytes"], 64)
				if err0 == nil && err1 == nil {
					mqtt.PublishValue(&dStorageUsedPercent, strconv.FormatFloat(usedBytes*100.0/totalBytes, 'f', 1, 64))
					mqtt.PublishValue(&dStorageUsed, strconv.FormatFloat(usedBytes/1024.0/1024.0/1024.0, 'f', 2, 64))
				} else {
					logrus.Warn(err0, err1)
				}
			}
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
		logrus.SetFormatter(&logrus.TextFormatter{DisableQuote: true, FullTimestamp: true})
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
