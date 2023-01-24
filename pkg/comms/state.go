package comms

import "strings"

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
)

type EntityCategory string

var (
	EC_DEFAULT    EntityCategory = ""
	EC_CONFIG                    = "config"
	EC_DIAGNOSTIC                = "diagnostic"
)

type DeviceClassType string

var (
	DC_MOTION DeviceClassType = "motion"
)

const (
	STATE_ON  = "on"
	STATE_OFF = "off"
)

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

func (s *Sensor) StateTopic() string {
	return s.Identifier + "/" + strings.ToLower(s.Name)
}
