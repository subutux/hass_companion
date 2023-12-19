package mobile_app

import (
	"fmt"
	"net/url"
	"time"

	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/hass/auth"
	"github.com/subutux/hass_companion/hass/mobile_app/sensors"
	"github.com/subutux/hass_companion/hass/rest"
	"github.com/subutux/hass_companion/hass/ws"
)

type MobileApp struct {
	credentials     *auth.Credentials
	Registration    *rest.RegistrationResponse
	ws              *ws.Client
	SensorCollector *sensors.Collector
}

func NewMobileApp(registration *rest.RegistrationResponse, creds *auth.Credentials, ws *ws.Client, interval time.Duration) *MobileApp {
	ma := MobileApp{
		credentials:  creds,
		Registration: registration,
		ws:           ws,
	}
	webhook, _ := ma.WebhookUrl()
	ma.SensorCollector = sensors.NewCollector(webhook, interval)

	return &ma
}

// WebhookUrl determines the correct URL to use for sending data back to
// Home assistant.
func (m *MobileApp) WebhookUrl() (string, error) {
	pp.Println(m.Registration, m.credentials)
	if m.Registration.CloudhookURL != "" {
		return m.Registration.CloudhookURL, nil
	}
	if m.Registration.RemoteUIURL != "" {
		url, err := url.Parse(m.Registration.RemoteUIURL)
		if err != nil {
			return "", err
		}
		url.Path = fmt.Sprintf("/api/webhook/%s", m.Registration.WebhookID)
		return url.String(), nil
	}

	url, err := url.Parse(m.credentials.Server)
	if err != nil {
		return "", err
	}
	url.Path = fmt.Sprintf("/api/webhook/%s", m.Registration.WebhookID)
	return url.String(), nil
}
