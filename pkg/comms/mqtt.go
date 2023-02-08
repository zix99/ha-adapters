package comms

import (
	"encoding/json"
	"errors"
	"time"

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
		mqtt: client,
		Qos:  2,
	}

	logrus.Info("Connected!")

	return ret, nil
}

func (s *Mqtt) Close() error {
	s.mqtt.Disconnect(1000)
	return nil
}

// Publish a topic with a string or []byte payload
func (s *Mqtt) Publish(topic string, payload []byte) error {
	logrus.Tracef("Publishing on %s: %s", topic, payload)
	ret := s.mqtt.Publish(topic, s.Qos, false, payload)
	err := resolveToken(ret)
	if err != nil {
		logrus.Debugf("Error publishing to %s: %s", topic, err)
	}
	return err
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

func (s *Mqtt) PublishState(device SensorTopic, state SensorState) {
	s.PublishString(device.StateTopic(), string(state))
}

func (s *Mqtt) PublishValue(device SensorTopic, value string) {
	s.PublishString(device.StateTopic(), value)
}

func (s *Mqtt) Subscribe(topic string) (events <-chan mqtt.Message, err error) {
	c := make(chan mqtt.Message, 10)
	logrus.Debugf("Subscribing to %s...", topic)
	t := s.mqtt.Subscribe(topic, 0, func(client mqtt.Client, m mqtt.Message) {
		c <- m
	})
	if err := resolveToken(t); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Mqtt) SubscribeFunc(topic string, f func(topic, val string)) error {
	logrus.Debugf("Subscribing to %s...", topic)
	t := s.mqtt.Subscribe(topic, 0, func(c mqtt.Client, m mqtt.Message) {
		go f(m.Topic(), string(m.Payload()))
	})
	return resolveToken(t)
}

func resolveToken(t mqtt.Token) error {
	if !t.WaitTimeout(5 * time.Second) {
		return errors.New("mqtt request timed out")
	}
	return t.Error()
}
