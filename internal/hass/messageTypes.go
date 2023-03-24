package hass

import "time"

type EventMessage struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Event struct {
		Data      interface{} `json:"data"`
		EventType string      `json:"event_type"`
		TimeFired time.Time   `json:"time_fired"`
		Origin    string      `json:"origin"`
		Context   struct {
			ID       string      `json:"id"`
			ParentID interface{} `json:"parent_id"`
			UserID   string      `json:"user_id"`
		} `json:"context"`
	} `json:"event"`
}

type MobileDeviceRegistration struct {
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
