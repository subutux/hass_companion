package rest

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
