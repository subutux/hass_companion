package rest

import (
	"github.com/go-resty/resty/v2"
	"github.com/subutux/hass_companion/hass/auth"
)

type Client struct {
	credentials *auth.Credentials
}

func NewClient(credentials *auth.Credentials) *Client {
	return &Client{
		credentials: credentials,
	}
}

func (c *Client) Api() *resty.Client {
	return resty.New().
		SetBaseURL(c.credentials.Server).
		SetAuthToken(c.credentials.AccessToken())
}
