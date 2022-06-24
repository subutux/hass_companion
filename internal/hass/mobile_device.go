package hass

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/matishsiao/goInfo"
	"github.com/subutux/hass_companion/internal/config"
)

type MobileDevice struct {
	CloudhookURL string `json:"cloudhook_url"`
	RemoteUIURL  string `json:"remote_ui_url"`
	Secret       string `json:"secret"`
	WebhookID    string `json:"webhook_id"`
	hass         *Hass
}

type RegisterPushNotification struct {
	ID             int    `json:"id"`
	Type           string `json:"type"`
	WebhookID      string `json:"webhook_id"`
	SupportConfirm bool   `json:"support_confirm"`
}

type PushNotification struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Event struct {
		Message string      `json:"message"`
		Title   string      `json:"title"`
		Data    interface{} `json:"data"`
	} `json:"event"`
}

func (h *Hass) RegisterCompanion() *MobileDevice {
	var mobileDevice MobileDevice
	mdc := config.Get("MobileDevice")
	log.Printf("trying to load saved mobileDevice: %s", mdc)
	if mdc != "" {
		err := json.Unmarshal([]byte(mdc), &mobileDevice)
		if err == nil {
			mobileDevice.hass = h
			return &mobileDevice
		}
		log.Printf("Failed to load configuration! %s", err)
	}
	OSInfo, err := goInfo.GetInfo()
	if err != nil {
		log.Fatalf("Unable to determine system: %s", err)
	}
	Registration := MobileDeviceRegistration{
		DeviceID:           uuid.NewString(),
		AppID:              "be.subutux.companion",
		AppName:            "HASS Companion",
		AppVersion:         "0.0.1",
		DeviceName:         OSInfo.Hostname,
		Manufacturer:       "unknown",
		Model:              "unknown",
		OsName:             OSInfo.OS,
		OsVersion:          OSInfo.Core,
		SupportsEncryption: false,
		AppData: struct {
			PushWebsocketChannel bool `json:"push_websocket_channel"`
		}{
			PushWebsocketChannel: true,
		},
	}

	data, err := json.Marshal(Registration)

	if err != nil {
		log.Fatalf("Unable to marshal Registration data: %v", err)
	}

	config.Set("MobileDeviceRegistration", string(data))
	resp, err := h.apiRequest().SetBody(data).SetResult(MobileDevice{}).
		Post("/api/mobile_app/registrations")
	if !resp.IsSuccess() {
		log.Fatalf("Unable to register device: %v %v", resp.Status(), resp.String())
	}
	mobileDevice = *resp.Result().(*MobileDevice)
	log.Printf("-- Got %s", resp.String())
	mobileDevice.hass = h
	data, err = json.Marshal(mobileDevice)

	if err != nil {
		log.Fatalf("Unable to marshal mobileDevice response data: %v", err)
	}

	config.Set("MobileDevice", string(data))
	return &mobileDevice
}

func (m *MobileDevice) SetupPush(f func(*PushNotification)) {
	id := m.hass.NextID()
	m.hass.Socket.Send(RegisterPushNotification{
		ID:             id,
		Type:           "mobile_app/push_notification_channel",
		WebhookID:      m.WebhookID,
		SupportConfirm: false,
	})
	m.hass.RegisterNotificationHandler(id, f)
}
