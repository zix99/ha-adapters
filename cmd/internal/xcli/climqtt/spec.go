package climqtt

import (
	"ha-adapters/pkg/comms"

	"github.com/urfave/cli/v2"
)

var Flags = []cli.Flag{
	&cli.StringFlag{
		Name:     "mqtt-uri",
		EnvVars:  []string{"MQTT_URI"},
		Usage:    "Set MQTT broker in format hostname:port",
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
	&cli.IntFlag{
		Name:  "qos",
		Usage: "Default MQTT QOS",
		Value: 0,
	},
}

func BuildClientFromFlags(c *cli.Context) (*comms.Mqtt, error) {
	var (
		uri      = c.String("mqtt-uri")
		username = c.String("username")
		password = c.String("password")
		qos      = c.Int("qos")
	)

	client, err := comms.NewMqtt(uri, username, password)
	if err != nil {
		return nil, err
	}
	client.Qos = byte(qos)

	return client, err
}
