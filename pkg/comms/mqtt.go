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

	storedSubs   map[string]func(mqtt.Message) // topic -> msg handler
	loopShutdown chan<- struct{}
}

var _ Publisher = &Mqtt{}

func NewMqtt(brokerUri string, username, password string) (*Mqtt, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerUri)
	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}

	client := &Mqtt{
		Qos:        2,
		storedSubs: make(map[string]func(mqtt.Message)),
	}

	opts.OnConnect = func(c mqtt.Client) {
		client.resubscribe()
	}

	opts.WillEnabled = true
	opts.WillTopic = TopicStatus
	opts.WillPayload = []byte(STATUS_OFFLINE)

	// NewClient makes a copy of `opts`
	client.mqtt = mqtt.NewClient(opts)

	logrus.Infof("Connecting to %s...", brokerUri)
	if err := resolveToken(client.mqtt.Connect()); err != nil {
		return nil, err
	}

	client.loopShutdown = client.startOnlineLoop(1 * time.Minute)

	logrus.Info("Connected!")

	return client, nil
}

func (s *Mqtt) Close() error {
	if s.loopShutdown != nil {
		s.loopShutdown <- struct{}{}
		s.loopShutdown = nil

		s.PublishString(TopicStatus, STATUS_OFFLINE)
	}
	s.mqtt.Disconnect(1000)
	return nil
}

// Publish a topic with a string or []byte payload
func (s *Mqtt) publish(topic string, retain bool, payload []byte) error {
	logrus.Tracef("Publishing on %s: %s", topic, payload)
	ret := s.mqtt.Publish(topic, s.Qos, retain, payload)
	err := resolveToken(ret)
	if err != nil {
		logrus.Warnf("Error publishing to %s: %s", topic, err)
	}
	return err
}

func (s *Mqtt) Publish(topic string, payload []byte) error {
	return s.publish(topic, false, payload)
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

func (s *Mqtt) RetainJson(topic string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.publish(topic, true, b)
}

func (s *Mqtt) PublishState(device SensorTopic, state SensorState) {
	s.PublishString(device.StateTopic(), string(state))
}

func (s *Mqtt) PublishValue(device SensorTopic, value string) {
	s.PublishString(device.StateTopic(), value)
}

func (s *Mqtt) Subscribe(topic string) (events <-chan mqtt.Message, err error) {
	c := make(chan mqtt.Message, 10)
	events = c
	err = s.subscribeStore(topic, func(m mqtt.Message) {
		c <- m
	})
	return
}

func (s *Mqtt) SubscribeFunc(topic string, f func(topic, val string)) error {
	return s.subscribeStore(topic, func(m mqtt.Message) {
		f(m.Topic(), string(m.Payload()))
	})
}

func (s *Mqtt) subscribeStore(topic string, f func(m mqtt.Message)) error {
	if err := s.subscribeInternal(topic, f); err != nil {
		return err
	}
	s.storedSubs[topic] = f
	return nil
}

func (s *Mqtt) resubscribe() {
	for topic, f := range s.storedSubs {
		s.subscribeInternal(topic, f)
	}
}

func (s *Mqtt) subscribeInternal(topic string, f func(m mqtt.Message)) error {
	logrus.Debugf("Subscribing to %s...", topic)
	t := s.mqtt.Subscribe(topic, 0, func(c mqtt.Client, m mqtt.Message) {
		go f(m)
	})
	if err := resolveToken(t); err != nil {
		logrus.Warnf("Error subscribing to %s: %s", topic, err)
		return err
	}
	return nil
}

func (s *Mqtt) startOnlineLoop(intv time.Duration) chan<- struct{} {
	shutdown := make(chan struct{})

	s.PublishString(TopicStatus, STATUS_ONLINE)

	go func() {
		ticker := time.NewTicker(intv)
		defer ticker.Stop()

		for {
			select {
			case <-shutdown:
				return
			case <-ticker.C:
				s.PublishString(TopicStatus, STATUS_ONLINE)
			}
		}
	}()

	return shutdown
}

func resolveToken(t mqtt.Token) error {
	if !t.WaitTimeout(5 * time.Second) {
		return errors.New("mqtt request timed out")
	}
	return t.Error()
}
