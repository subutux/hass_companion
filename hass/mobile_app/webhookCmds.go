package mobile_app

import "encoding/json"

type WebhookCmd struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

func NewWebhookGetConfigCmd() string {
	data, _ := json.Marshal(WebhookCmd{
		Type: "get_config",
	})

	return string(data)
}
