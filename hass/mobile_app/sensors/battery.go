package sensors

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

type Battery struct {
	Sensor
	conn     *dbus.Conn
	dbusPath dbus.ObjectPath
}

func (b *Battery) GetSensors() []*Sensor {
	b.Update()
	return []*Sensor{&b.Sensor}
}

func (b *Battery) Enable() {
	b.Sensor.Disabled = false
}

func (b *Battery) Disable() {
	b.Sensor.Disabled = true
}

func (b *Battery) Update() error {
	if strings.HasSuffix(b.Name, "_level") {
		variant, err := b.conn.Object("org.freedesktop.UPower", b.dbusPath).
			GetProperty("org.freedesktop.UPower.Device.Percentage")
		if err != nil {
			return err
		}
		b.State = int(variant.Value().(float64))
	} else if strings.HasSuffix(b.Name, "_state") {
		variant, err := b.conn.Object("org.freedesktop.UPower", b.dbusPath).
			GetProperty("org.freedesktop.UPower.Device.State")
		if err != nil {
			return err
		}
		state := variant.Value().(uint32)
		charging := false
		if state == 1 {
			charging = true
		}
		b.State = charging
	}

	return nil
}

func NewBatteryLevel(systemdbus *dbus.Conn, path dbus.ObjectPath, name string, state int, disabled bool) *Battery {
	name = name + "_level"

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
		conn:     systemdbus,
		dbusPath: path,
	}
}

func NewBatteryState(systemdbus *dbus.Conn, path dbus.ObjectPath, name string, state bool, disabled bool) *Battery {
	name = name + "_state"
	return &Battery{
		Sensor: Sensor{
			Name:           name,
			UniqueID:       name,
			DeviceClass:    "battery_charging",
			State:          strconv.FormatBool(state),
			Type:           "binary_sensor",
			EntityCategory: "diagnostic",
			Disabled:       disabled,
		},
		conn:     systemdbus,
		dbusPath: path,
	}
}

func DiscoverBatteries(systemdbus *dbus.Conn) ([]*Battery, error) {
	s := []dbus.ObjectPath{}
	batteries := []*Battery{}

	err := systemdbus.Object("org.freedesktop.UPower", "/org/freedesktop/UPower").
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
		o := systemdbus.Object("org.freedesktop.UPower", path)
		deviceType, err := o.GetProperty("org.freedesktop.UPower.Device.Type")
		if err != nil {
			return batteries, err
		}
		if deviceType.Value().(uint32) == 2 {
			variant, err := o.GetProperty("org.freedesktop.UPower.Device.Percentage")
			if err != nil {
				return batteries, err
			}
			percentage = int(variant.Value().(float64))

			variant, err = o.GetProperty("org.freedesktop.UPower.Device.State")
			if err != nil {
				return batteries, err
			}
			state := variant.Value().(uint32)
			charging := false
			if state == 1 {
				charging = true
			}

			batteries = append(batteries, NewBatteryLevel(systemdbus, path, name, percentage, !present))
			batteries = append(batteries, NewBatteryState(systemdbus, path, name, charging, !present))

		}
	}

	return batteries, nil
}
