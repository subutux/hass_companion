package rest

import "encoding/json"

type RegistrationResponse struct {
	CloudhookURL string `json:"cloudhook_url"`
	RemoteUIURL  string `json:"remote_ui_url"`
	Secret       string `json:"secret"`
	WebhookID    string `json:"webhook_id"`
}

func (c *Client) RegisterMobileApp(registration interface{}) (*RegistrationResponse, error) {
	response, err := c.Api().R().
		SetBody(registration).
		SetResult(&RegistrationResponse{}).
		Post("/api/mobile_app/registrations")
	return response.Result().(*RegistrationResponse), err
}

func (c *Client) GetConfig(webhookID string) (string, error) {
	return c.SendCmd(webhookID, NewWebhookGetConfigCmd())
}
func (c *Client) SendCmd(webhookID string, cmd WebhookCmd) (string, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return "", err
	}
	response, err := c.Api().R().
		SetBody(string(data)).
		Post("/api/webhook/" + webhookID)

	return string(response.Body()), err
}
