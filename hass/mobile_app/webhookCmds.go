package mobile_app

import "encoding/json"

type WebhookCmd struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type GeoData struct {
	Gps              []float64 `json:"gps,omitempty"`
	GpsAccuracy      int       `json:"gps_accuracy,omitempty"`
	Battery          int       `json:"battery,omitempty"`
	Speed            int       `json:"speed,omitempty"`
	Altitude         int       `json:"altitude,omitempty"`
	Course           int       `json:"course,omitempty"`
	VerticalAccuracy int       `json:"vertical_accuracy,omitempty"`
}

func NewWebhookGetConfigCmd() string {
	data, _ := json.Marshal(WebhookCmd{
		Type: "get_config",
	})

	return string(data)
}

func NewWebhookUpdateLocationCmd(location *Location) string {

	data, _ := json.Marshal(WebhookCmd{
		Type: "update_location",
		Data: GeoData{
			Gps:         []float64{location.Latitude, location.Longitude},
			GpsAccuracy: int(location.Accuracy),
		},
	})

	return string(data)
}
