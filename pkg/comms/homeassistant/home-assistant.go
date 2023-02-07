package homeassistant

import (
	"ha-adapters/pkg/comms"
	"ha-adapters/pkg/stemplate"
	"path"
	"time"

	"golang.org/x/exp/maps"
)

var (
	Default_HA_Root   = "homeassistant"
	Default_HA_Prefix = "ha-adapters-"
	Default_HA_Via    = "ha-adapters"
)

var (
	topic_advertise_config = stemplate.MustNew("{{.HA.TopicRoot}}/{{.Dev.Type}}/{{.HA.TopicPrefix}}{{.Dev.Identifier}}/{{.Dev.SanitizedName}}/config")
)

type HomeAssistant struct {
	mqtt        *comms.Mqtt
	TopicRoot   string
	TopicPrefix string

	statusShutdown chan<- struct{}
}

func NewHomeAssistant(mqtt *comms.Mqtt) (*HomeAssistant, error) {
	ha := &HomeAssistant{
		mqtt:        mqtt,
		TopicRoot:   Default_HA_Root,
		TopicPrefix: Default_HA_Prefix,
	}

	shutdown, err := ha.startOnlineLoop(1 * time.Minute)
	if err != nil {
		return nil, err
	}
	ha.statusShutdown = shutdown

	return ha, nil
}

func (s *HomeAssistant) Close() error {
	if s.statusShutdown != nil {
		s.statusShutdown <- struct{}{}
		s.statusShutdown = nil

		s.mqtt.PublishString(s.TopicStatus(), comms.STATUS_OFFLINE)
	}
	return nil
}

func (s *HomeAssistant) Advertise(d *comms.Sensor) error {
	topicData := struct {
		HA  *HomeAssistant
		Dev *comms.Sensor
	}{s, d}

	topic := topic_advertise_config.Execute(topicData)

	payload := s.deviceBaseConfig(&d.DeviceClass)
	maps.Copy(payload, JsonMap{
		"state_topic": d.StateTopic(),
		"name":        d.DeviceClass.DeviceName + " " + d.Name,
		"unique_id":   d.UniqueId(),
	})

	switch d.Type {
	case comms.DT_BINARY_SENSOR:
		maps.Copy(payload, JsonMap{
			"payload_on":  comms.STATE_ON,
			"payload_off": comms.STATE_OFF,
		})
	case comms.DT_SWITCH:
		payload["command_topic"] = d.StateTopic()
		payload["optimistic"] = true
	case comms.DT_SENSOR:
		payload["unit_of_measurement"] = d.UnitOfMeasurement
	}

	// Optional classes
	if d.ClassType != "" {
		payload["device_class"] = d.ClassType
	}
	if d.Category != "" {
		payload["entity_category"] = d.Category
	}
	if d.Icon != "" {
		payload["icon"] = d.Icon
	}

	// Publish!
	return s.mqtt.PublishJson(topic, payload)
}

func (s *HomeAssistant) deviceBaseConfig(dc *comms.DeviceClass) JsonMap {
	return JsonMap{
		"availability_topic": s.TopicStatus(),
		"qos":                s.mqtt.Qos,
		"device": JsonMap{
			"name":         dc.DeviceName,
			"manufacturer": dc.Manufacturer,
			"model":        dc.Model,
			"identifiers":  dc.Identifier,
			"sw_version":   dc.Version,
			"via_device":   Default_HA_Via,
		},
	}
}

func (s *HomeAssistant) TopicStatus() string {
	return path.Join(comms.TopicPrefix, "status")
}

func (s *HomeAssistant) startOnlineLoop(intv time.Duration) (chan<- struct{}, error) {
	shutdown := make(chan struct{})

	if err := s.mqtt.PublishString(s.TopicStatus(), comms.STATUS_ONLINE); err != nil {
		return nil, err
	}

	go func() {
		ticker := time.NewTicker(intv)
		defer ticker.Stop()

		for {
			select {
			case <-shutdown:
				return
			case <-ticker.C:
				s.mqtt.PublishString(s.TopicStatus(), comms.STATUS_ONLINE)
			}
		}
	}()

	return shutdown, nil
}
