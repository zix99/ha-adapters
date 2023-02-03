package comms

import (
	"path"
	"regexp"
	"strings"
)

var TopicPrefix = "ha-adapters"

type DeviceClass struct {
	DeviceName   string // A device name
	Manufacturer string // Manufacturer of device
	Model        string // Model of device
	Identifier   string // eg serial number
	Version      string // Software version
}

type DeviceType string

const (
	DT_BINARY_SENSOR DeviceType = "binary_sensor"
	DT_SENSOR        DeviceType = "sensor"
	DT_SWITCH        DeviceType = "switch"
)

type EntityCategory string

var (
	EC_DEFAULT    EntityCategory = ""
	EC_CONFIG     EntityCategory = "config"
	EC_DIAGNOSTIC EntityCategory = "diagnostic"
)

type DeviceClassType string

var (
	DC_MOTION  DeviceClassType = "motion"
	DC_BATTERY DeviceClassType = "battery"
)

type DeviceState string

const (
	STATE_ON  DeviceState = "on"
	STATE_OFF DeviceState = "off"
)

func StateStr(on bool) DeviceState {
	if on {
		return STATE_ON
	}
	return STATE_OFF
}

func StrState(s string) bool {
	return strings.EqualFold(s, string(STATE_ON))
}

const (
	STATUS_ONLINE  = "online"
	STATUS_OFFLINE = "offline"
)

type Sensor struct {
	DeviceClass
	Name string
	Type DeviceType
	Icon string

	UnitOfMeasurement string
	Category          EntityCategory
	ClassType         DeviceClassType
}

type DeviceStateTopic interface {
	StateTopic() string
}

func (s *DeviceClass) NewBinarySensor(name string) *Sensor {
	return &Sensor{
		DeviceClass: *s,
		Name:        name,
		Type:        DT_BINARY_SENSOR,
	}
}

func (s *Sensor) SanitizedName() string {
	return sanitize(s.Name)
}

func (s *Sensor) UniqueId() string {
	return sanitize(s.Identifier) + "." + sanitize(s.Name)
}

func (s *Sensor) StateTopic() string {
	return path.Join(
		TopicPrefix,
		sanitize(s.Identifier),
		sanitize(s.Name))
}

var santizeRegex = regexp.MustCompile(`[^a-zA-Z0-9\-]+`)

func sanitize(s string) string {
	sanitized := santizeRegex.ReplaceAllString(s, "_")
	sanitized = strings.Trim(sanitized, "_")
	sanitized = strings.ToLower(sanitized)
	return sanitized
}
