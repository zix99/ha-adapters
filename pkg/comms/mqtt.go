package comms

import (
	"encoding/json"
	"path"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

type Publisher interface {
	Publish(topic string, payload []byte) error
	PublishString(topic string, payload string) error
}

type Mqtt struct {
	mqtt mqtt.Client
	Qos  byte
}

var _ Publisher = &Mqtt{}

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
	}

	if err := ret.PublishString(ret.TopicStatus(), STATUS_ONLINE); err != nil {
		client.Disconnect(1000)
		return nil, err
	}

	logrus.Info("Connected!")

	return ret, nil
}

func (s *Mqtt) Close() error {
	err := s.PublishString(s.TopicStatus(), STATUS_OFFLINE)
	s.mqtt.Disconnect(1000)
	return err
}

func (s *Mqtt) TopicStatus() string {
	return path.Join(TopicPrefix, "status")
}

// Publish a topic with a string or []byte payload
func (s *Mqtt) Publish(topic string, payload []byte) error {
	logrus.Tracef("Publishing on %s: %s", topic, payload)
	ret := s.mqtt.Publish(topic, s.Qos, false, payload)
	return resolveToken(ret)
}

func (s *Mqtt) PublishString(topic string, payload string) error {
	return s.Publish(topic, []byte(payload))
}

func (s *Mqtt) PublishJson(topic string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.Publish(topic, b)
}

func (s *Mqtt) PublishState(device DeviceStateTopic, state string) {
	s.PublishString(device.StateTopic(), state)
}

func resolveToken(t mqtt.Token) error {
	t.Wait()
	return t.Error()
}
