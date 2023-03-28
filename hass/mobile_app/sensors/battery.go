package sensors

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/godbus/dbus"
)

type Battery struct {
	Sensor
}

func NewBattery(name string, state int, disabled bool) *Battery {
	return &Battery{
		Sensor: Sensor{
			Name:              name,
			UniqueID:          name,
			DeviceClass:       "battery",
			Icon:              "mdi:battery",
			State:             strconv.Itoa(state),
			StateClass:        "measurement",
			Type:              "sensor",
			UnitOfMeasurement: "%",
			EntityCategory:    "diagnostic",
			Disabled:          disabled,
		},
	}
}

func DiscoverBatteries() ([]*Battery, error) {
	s := []dbus.ObjectPath{}
	batteries := []*Battery{}

	conn, err := dbus.SystemBus()
	if err != nil {
		return batteries, err
	}
	err = conn.Object("org.freedesktop.UPower", "/org/freedesktop/UPower").
		Call("org.freedesktop.UPower.EnumerateDevices", 0).
		Store(&s)

	if err != nil {
		return batteries, err
	}

	for _, path := range s {
		split := strings.Split(fmt.Sprint(path), "/")
		name := split[len(split)-1]
		percentage := 0
		present := true
		variant, err := conn.Object("org.freedesktop.UPower", path).
			GetProperty("org.freedesktop.UPower.Device.Percentage")
		if err != nil {
			return batteries, err
		}
		percentage = int(variant.Value().(float64))

		batteries = append(batteries, NewBattery(name, percentage, !present))

	}

	return batteries, nil
}
