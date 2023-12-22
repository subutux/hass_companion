package sensors

import "github.com/prometheus/procfs"

type AverageLoad struct {
	Sensor
}

func (a *AverageLoad) GetSensors() []*Sensor {
	a.Update()
	return []*Sensor{&a.Sensor}
}

func (a *AverageLoad) Enable() {
	a.Sensor.Disabled = false
}

func (a *AverageLoad) Disable() {
	a.Sensor.Disabled = true
}

func (a *AverageLoad) Update() (err error) {
	fs, err := procfs.NewDefaultFS()

	if err != nil {
		return err
	}

	avg, err := fs.LoadAvg()
	if err != nil {
		return err
	}
	a.State = avg.Load1
	a.Attributes = map[string]any{
		"load_5":  avg.Load5,
		"load_15": avg.Load15,
	}
	return nil
}

func DiscoverAverageLoad() (*AverageLoad, error) {

	sensor := AverageLoad{
		Sensor{
			Name:           "load",
			UniqueID:       "load",
			Type:           "sensor",
			Icon:           "mdi:cpu-64-bit",
			EntityCategory: "diagnostic",
			Disabled:       false,
		},
	}

	err := sensor.Update()
	if err != nil {
		return nil, err
	}
	return &sensor, nil
}
