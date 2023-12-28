package rest

import "encoding/json"

type RegistrationResponse struct {
	CloudhookURL string `json:"cloudhook_url"`
	RemoteUIURL  string `json:"remote_ui_url"`
	Secret       string `json:"secret"`
	WebhookID    string `json:"webhook_id"`
}
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
func (c *Client) RegisterMobileApp(registration interface{}) (*RegistrationResponse, error) {
	response, err := c.Api().R().
		SetBody(registration).
		SetResult(&RegistrationResponse{}).
		Post("/api/mobile_app/registrations")
	return response.Result().(*RegistrationResponse), err
}

func (c *Client) GetConfig(webhookID string) (string, error) {

	response, err := c.Api().R().
		SetBody(NewWebhookGetConfigCmd()).
		Post("/api/webhook/" + webhookID)

	return string(response.Body()), err
}
