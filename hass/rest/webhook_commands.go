package rest

type WebhookCmd struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

func NewWebhookGetConfigCmd() WebhookCmd {
	return WebhookCmd{
		Type: "get_config",
	}
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
