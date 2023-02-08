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

type SensorType string

const (
	ST_BINARY_SENSOR SensorType = "binary_sensor"
	ST_SENSOR        SensorType = "sensor"
	ST_SWITCH        SensorType = "switch"
)

type SensorCategory string

var (
	EC_DEFAULT    SensorCategory = ""
	EC_CONFIG     SensorCategory = "config"
	EC_DIAGNOSTIC SensorCategory = "diagnostic"
)

type SensorClassType string

var (
	SC_MOTION  SensorClassType = "motion"
	SC_BATTERY SensorClassType = "battery"
)

type SensorState string

const (
	STATE_ON  SensorState = "on"
	STATE_OFF SensorState = "off"
)

func StateStr(on bool) SensorState {
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
	Name     string
	Type     SensorType
	Icon     string
	JsonPath string

	UnitOfMeasurement string
	Category          SensorCategory
	ClassType         SensorClassType

	Extra map[string]interface{}
}

type SensorTopic interface {
	StateTopic() string
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
