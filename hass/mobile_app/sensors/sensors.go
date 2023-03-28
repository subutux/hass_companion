package sensors

type Sensor struct {
	Attributes        interface{} `json:"attributes"`
	DeviceClass       string      `json:"device_class"`
	Icon              string      `json:"icon"`
	Name              string      `json:"name"`
	State             string      `json:"state"`
	Type              string      `json:"type"`
	UniqueID          string      `json:"unique_id"`
	UnitOfMeasurement string      `json:"unit_of_measurement"`
	StateClass        string      `json:"state_class"`
	EntityCategory    string      `json:"entity_category"`
	Disabled          bool        `json:"disabled"`
}
