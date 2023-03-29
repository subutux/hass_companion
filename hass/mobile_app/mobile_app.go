package mobile_app

import (
	"fmt"
    "log"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/subutux/hass_companion/hass/auth"
	"github.com/subutux/hass_companion/hass/mobile_app/sensors"
	"github.com/subutux/hass_companion/hass/rest"
	"github.com/subutux/hass_companion/hass/ws"
)

type SensorRegistration struct {
	Sensor *sensors.Sensor `json:"data"`
	Type   string          `json:"type"`
}

type SensorUpdates struct {
	Sensors []*sensors.Sensor `json:"data"`
	Type    string            `json:"type"`
}

func NewSensorRegistration(sensor *sensors.Sensor) *SensorRegistration {
	return &SensorRegistration{
		Sensor: sensor,
		Type:   "register_sensor",
	}
}

func NewSensorUpdates(sensors []*sensors.Sensor) *SensorUpdates {
	return &SensorUpdates{
		Sensors: sensors,
		Type:    "update_sensor_states",
	}
}

type MobileAppRegistration struct {
	DeviceID           string `json:"device_id"`
	AppID              string `json:"app_id"`
	AppName            string `json:"app_name"`
	AppVersion         string `json:"app_version"`
	DeviceName         string `json:"device_name"`
	Manufacturer       string `json:"manufacturer"`
	Model              string `json:"model"`
	OsName             string `json:"os_name"`
	OsVersion          string `json:"os_version"`
	SupportsEncryption bool   `json:"supports_encryption"`
	AppData            struct {
		PushWebsocketChannel bool `json:"push_websocket_channel"`
	} `json:"app_data"`
}

func NewMobileAppRegistration() *MobileAppRegistration {
	OSInfo, err := host.Info()
	if err != nil {
		log.Fatalf("Unable to determine system: %s", err)
	}
	info, err := GetServerInformation()
	if err != nil {
		log.Fatalf("Unable to determine system: %s", err)
	}
	return &MobileAppRegistration{
		DeviceID:           OSInfo.HostID,
		AppID:              "be.subutux.companion",
		AppName:            "HASS Companion",
		AppVersion:         "0.0.1",
		DeviceName:         OSInfo.Hostname,
		Manufacturer:       info.Vendor,
		Model:              info.Name + " " + info.Version,
		OsName:             OSInfo.Platform,
		OsVersion:          OSInfo.PlatformVersion,
		SupportsEncryption: false,
		AppData: struct {
			PushWebsocketChannel bool `json:"push_websocket_channel"`
		}{
			PushWebsocketChannel: true,
		},
	}
}

type MobileApp struct {
	credentials  *auth.Credentials
	Registration *rest.RegistrationResponse
	ws           *ws.Client
}

func NewMobileApp(registration *rest.RegistrationResponse, ws *ws.Client) *MobileApp {
	return &MobileApp{
		Registration: registration,
		ws:           ws,
	}
}

// WebhookUrl determines the correct URL to use for sending data back to
// Home assistant.
func (m *MobileApp) WebhookUrl() (string, error) {
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

func (m *MobileApp) RegisterSensor(sensor *sensors.Sensor) ([]byte, error) {
	webhook, err := m.WebhookUrl()
	if err != nil {
		return nil, err
	}
	r, err := resty.New().R().
		SetBody(NewSensorRegistration(sensor)).
		Post(webhook)
	if err != nil {
		return nil, err
	}

	return r.Body(), err
}

func (m *MobileApp) UpdateSensors(sensors []*sensors.Sensor) ([]byte, error) {
	webhook, err := m.WebhookUrl()
	if err != nil {
		return nil, err
	}
	r, err := resty.New().R().
		SetBody(NewSensorUpdates(sensors)).
		Post(webhook)
	if err != nil {
		return nil, err
	}

	return r.Body(), err
}

// EnableWebsocketPushNotifications sends a command to Home Assistant that
// This client supports Push notifications over websockets.
func (m *MobileApp) EnableWebsocketPushNotifications() {
	m.ws.SendCommand(ws.NewSubscribeToPushNotificationsChannelCmd(m.Registration.WebhookID))
}

// WatchForPushNotifications calls the callback function with the notification
// whenever there is a push notification received on the websocket connection.
func (m *MobileApp) WatchForPushNotifications(callback func(notification *ws.IncomingPushNotificationMessage)) {
	for notification := range m.ws.PushNotificationChannel {
		callback(notification)
	}
}
