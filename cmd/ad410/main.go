package main

import (
	"ha-adapters/cmd/internal/xcli"
	"ha-adapters/cmd/internal/xcli/clilog"
	"ha-adapters/cmd/internal/xcli/climqtt"
	"ha-adapters/pkg/amcrest"
	"ha-adapters/pkg/comms"
	"ha-adapters/pkg/comms/homeassistant"
	"ha-adapters/pkg/stemplate"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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
		mediaDirTmpl    = mustCompileTemplateOrNil(c.String("media-dir"))
	)

	// setup and connect to doorbell
	doorbell, err := amcrest.ConnectAmcrest(amcrestUrl, amcrestUsername, amcrestPassword)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Println("Serial: " + doorbell.SerialNumber)
	logrus.Println("Type  : " + doorbell.DeviceType)
	logrus.Println(doorbell.SoftwareVersion)

	// Setup MQTT and eventing
	mqtt, err := climqtt.BuildClientFromFlags(c)
	if err != nil {
		logrus.Fatal(err)
	}
	defer mqtt.Close()

	ha, err := homeassistant.NewHomeAssistant(mqtt)
	if err != nil {
		mqtt.Close()
		logrus.Fatal(err)
	}
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
		Category:          comms.EC_DIAGNOSTIC,
	}

	dStorageUsed := comms.Sensor{
		DeviceClass:       device,
		Type:              comms.DT_SENSOR,
		Name:              "Storage Used",
		UnitOfMeasurement: "GB",
		Category:          comms.EC_DIAGNOSTIC,
	}

	dLightSwitch := comms.Sensor{
		DeviceClass: device,
		Type:        comms.DT_SWITCH,
		Category:    comms.EC_CONFIG,
		Name:        "Light",
	}

	ha.Advertise(&dButton)
	mqtt.PublishState(&dButton, comms.STATE_OFF)

	ha.Advertise(&dHuman)
	mqtt.PublishState(&dHuman, comms.STATE_OFF)

	ha.Advertise(&dMotion)
	mqtt.PublishState(&dMotion, comms.STATE_OFF)

	ha.Advertise(&dLightSwitch)
	mqtt.PublishState(&dLightSwitch, comms.STATE_OFF)

	ha.Advertise(&dStorageUsedPercent)
	ha.Advertise(&dStorageUsed)

	// Config/events
	mqtt.SubscribeFunc(dLightSwitch.StateTopic(), func(topic, val string) {
		doorbell.SetLight(comms.StrState(val))
	})

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
			case "NewFile":
				camPath := gjson.Get(event.Data, "File").String()
				// There are also `.mp4`, but they seem poorly encoded. I currently can't
				// get them to decode (moov atom error)
				if mediaDirTmpl != nil && strings.EqualFold(filepath.Ext(camPath), ".jpg") {
					outDir := mediaDirTmpl.Execute(struct{}{})
					os.MkdirAll(filepath.Dir(outDir), 0770)
					go doorbell.DownloadFileTo(camPath, outDir)
				}
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

func mustCompileTemplateOrNil(text string) *stemplate.STemplate {
	if text == "" {
		return nil
	}
	t, err := stemplate.New(text)
	if err != nil {
		logrus.Fatal(err)
	}
	return t
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{DisableQuote: true, FullTimestamp: true})

	app := cli.NewApp()
	app.Usage = "Amcrest AD410 to MQTT (Home-assistant)"
	app.Flags = xcli.JoinFlags(climqtt.Flags, []cli.Flag{
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
			Name:    "media-dir",
			Usage:   "Path to write media to. Uses path template. If empty, don't write",
			EnvVars: []string{"MEDIA_DIR"},
		},
	})
	clilog.AdaptForLogSettings(app)
	app.Action = runAD410

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
