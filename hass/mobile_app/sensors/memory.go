package sensors

import (
	"math"

	"github.com/prometheus/procfs"
)

type Memory struct {
	Sensor
}

func (a *Memory) GetSensors() []*Sensor {
	a.Update()
	return []*Sensor{&a.Sensor}
}

func (a *Memory) Enable() {
	a.Sensor.Disabled = false
}

func (a *Memory) Disable() {
	a.Sensor.Disabled = true
}

func (a *Memory) Update() (err error) {
	fs, err := procfs.NewDefaultFS()

	if err != nil {
		return err
	}

	mem, err := fs.Meminfo()
	if err != nil {
		return err
	}
	percentage := float64(*mem.MemTotal) / float64(*mem.MemFree)
	a.State = math.Ceil(percentage*100) / 100
	a.Attributes = map[string]any{
		"free":      mem.MemFree,
		"total":     mem.MemTotal,
		"available": mem.MemAvailable,
	}
	return nil
}

func DiscoverMemory() (*Memory, error) {

	sensor := Memory{
		Sensor{
			UnitOfMeasurement: "%",
			Icon:              "mdi:memory",
			Name:              "memory Usage",
			UniqueID:          "memory_usage",
			Type:              "sensor",
			EntityCategory:    "diagnostic",
			Disabled:          false,
		},
	}

	err := sensor.Update()
	if err != nil {
		return nil, err
	}
	return &sensor, nil
}
