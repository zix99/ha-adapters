package comms

import (
	"encoding/json"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

type Mqtt struct {
	mqtt      mqtt.Client
	Qos       byte
	TopicRoot string
}

func NewMqtt(brokerUri string, username, password string) (*Mqtt, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerUri)
	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}

	client := mqtt.NewClient(opts)

	logrus.Infof("Connecting to %s...", brokerUri)
	if err := resolveToken(client.Connect()); err != nil {
		return nil, err
	}

	ret := &Mqtt{
		client,
		2,
		"ha-adapters/",
	}

	if err := ret.PublishRaw(ret.TopicStatus(), STATUS_ONLINE); err != nil {
		client.Disconnect(1000)
		return nil, err
	}

	logrus.Info("Connected!")

	return ret, nil
}

func (s *Mqtt) Close() error {
	err := s.PublishRaw(s.TopicStatus(), STATUS_OFFLINE)
	s.mqtt.Disconnect(1000)
	return err
}

func (s *Mqtt) TopicStatus() string {
	return s.TopicRoot + "status"
}

// Publish a topic with a string or []byte payload
func (s *Mqtt) PublishRaw(topic string, payload interface{}) error {
	logrus.Debugf("Publishing on %s: %s", topic, payload)
	ret := s.mqtt.Publish(topic, s.Qos, false, payload)
	return resolveToken(ret)
}

func (s *Mqtt) PublishJson(topic string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.PublishRaw(topic, b)
}

func (s *Mqtt) PublishState(device DeviceStateTopic, state string) error {
	subtopic := strings.ToLower(device.StateTopic())
	return s.PublishRaw(s.TopicRoot+subtopic, state)
}

func resolveToken(t mqtt.Token) error {
	t.Wait()
	return t.Error()
}
