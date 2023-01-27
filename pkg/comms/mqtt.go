package comms

import (
	"encoding/json"
	"path"
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

	shutdown chan<- struct{}
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

	shutdown, err := ret.startOnlineLoop(1 * time.Minute)
	if err != nil {
		return nil, err
	}
	ret.shutdown = shutdown

	logrus.Info("Connected!")

	return ret, nil
}

func (s *Mqtt) Close() error {
	s.shutdown <- struct{}{}
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

func (s *Mqtt) PublishState(device DeviceStateTopic, state DeviceState) {
	s.PublishString(device.StateTopic(), string(state))
}

func (s *Mqtt) PublishValue(device DeviceStateTopic, value string) {
	s.PublishString(device.StateTopic(), value)
}

func (s *Mqtt) startOnlineLoop(intv time.Duration) (chan<- struct{}, error) {
	shutdown := make(chan struct{})

	if err := s.PublishString(s.TopicStatus(), STATUS_ONLINE); err != nil {
		s.mqtt.Disconnect(1000)
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
				s.PublishString(s.TopicStatus(), STATUS_ONLINE)
			}
		}
	}()

	return shutdown, nil
}

func resolveToken(t mqtt.Token) error {
	t.Wait()
	return t.Error()
}
