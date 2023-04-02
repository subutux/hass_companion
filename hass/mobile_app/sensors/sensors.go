package sensors

import (
	"log"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/k0kubun/pp/v3"
)

type Sensor struct {
	Attributes        map[string]any `json:"attributes,omitempty"`
	DeviceClass       string         `json:"device_class,omitempty"`
	Icon              string         `json:"icon,omitempty"`
	Name              string         `json:"name"`
	State             any            `json:"state"`
	Type              string         `json:"type"`
	UniqueID          string         `json:"unique_id"`
	UnitOfMeasurement string         `json:"unit_of_measurement,omitempty"`
	StateClass        string         `json:"state_class,omitempty"`
	EntityCategory    string         `json:"entity_category,omitempty"`
	Disabled          bool           `json:"disabled,omitempty"`
}

type SensorInterface interface {
	GetSensors() []*Sensor
	Disable()
	Enable()
}

type SensorUpdate struct {
	Attributes map[string]any `json:"attributes,omitempty"`
	Icon       string         `json:"icon,omitempty"`
	State      any            `json:"state"`
	Type       string         `json:"type"`
	UniqueID   string         `json:"unique_id"`
}

func NewSensorUpdateFromSensor(sensor *Sensor) *SensorUpdate {
	return &SensorUpdate{
		Attributes: sensor.Attributes,
		Icon:       sensor.Icon,
		State:      sensor.State,
		Type:       sensor.Type,
		UniqueID:   sensor.UniqueID,
	}
}

type SensorRegistration struct {
	Sensor *Sensor `json:"data"`
	Type   string  `json:"type"`
}

type SensorUpdates struct {
	Sensors []*SensorUpdate `json:"data"`
	Type    string          `json:"type"`
}

func NewSensorRegistration(sensor *Sensor) *SensorRegistration {
	return &SensorRegistration{
		Sensor: sensor,
		Type:   "register_sensor",
	}
}

func NewSensorUpdates(sensors []*SensorUpdate) *SensorUpdates {
	return &SensorUpdates{
		Sensors: sensors,
		Type:    "update_sensor_states",
	}
}

type Collector struct {
	mu       sync.Mutex
	StopChan chan struct{}
	Sensors  []SensorInterface
	// List containing the Unique IDs of registered sensors
	RegisteredSensors []string
	// List containing the Unique IDs of Disabled sensors
	DisabledSensors []string
	Interval        *time.Ticker
	Webhook         string
}

func NewCollector(webhook string, interval time.Duration) *Collector {
	return &Collector{
		mu:       sync.Mutex{},
		StopChan: make(chan struct{}),
		Interval: time.NewTicker(interval),
		Webhook:  webhook,
	}
}

func (c *Collector) IsRegistered(sensor *Sensor) bool {
	for _, id := range c.RegisteredSensors {
		if id == sensor.UniqueID {
			return true
		}
	}
	return false
}

func (c *Collector) IsDisabled(sensor *Sensor) bool {
	for _, id := range c.DisabledSensors {
		if id == sensor.UniqueID {
			return true
		}
	}
	return false
}

// HandleUpdateResponse checks if sensors are disabled
func (c *Collector) HandleUpdateResponse(response map[string]any) {
	for id, r := range response {
		for v, val := range r.(map[string]any) {
			if v == "is_disabled" {
				if val.(bool) {
					c.DisabledSensors = append(c.DisabledSensors, id)
				}
			}
		}
	}
}

func (c *Collector) AddSensors(sensors ...SensorInterface) {
	c.mu.Lock()
	c.Sensors = append(c.Sensors, sensors...)
	c.mu.Unlock()
}

func (c *Collector) AddSensor(sensor SensorInterface) {
	c.AddSensors(sensor)
}

func (c *Collector) Collect() {
	for {
		select {
		case <-c.StopChan:
			c.Interval.Stop()
			return
		case <-c.Interval.C:
			log.Printf("Collecting sensors...")
			var toUpdate []*SensorUpdate
			for _, _sensors := range c.Sensors {
				s := _sensors.GetSensors()
				for _, sensor := range s {
					log.Printf("Collecting sensor: %v", sensor.UniqueID)
					if !c.IsRegistered(sensor) {
						res, err := c.RegisterSensor(sensor)
						pp.Println(string(res), err)
						c.RegisteredSensors = append(c.RegisteredSensors, sensor.UniqueID)
					}
					if !c.IsDisabled(sensor) {
						toUpdate = append(toUpdate, NewSensorUpdateFromSensor(sensor))
					}
				}
			}
			pp.Println(toUpdate)
			res, err := c.UpdateSensors(toUpdate)
			pp.Println(string(res), err)
		}
	}
}

func (c *Collector) RegisterSensor(sensor *Sensor) ([]byte, error) {
	reg := NewSensorRegistration(sensor)
	pp.Println(reg)
	r, err := resty.New().R().
		SetBody(reg).
		Post(c.Webhook)
	if err != nil {
		return nil, err
	}
	return r.Body(), err
}

func (c *Collector) UpdateSensors(sensors []*SensorUpdate) ([]byte, error) {

	r, err := resty.New().R().
		SetBody(NewSensorUpdates(sensors)).
		Post(c.Webhook)
	if err != nil {
		return nil, err
	}

	return r.Body(), err
}
