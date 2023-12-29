package mobile_app

import (
	"os"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/subutux/hass_companion/internal/logger"
)

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
		logger.I().Error("Unable to determine system", "error", err)
		os.Exit(1)
	}
	info, err := GetServerInformation()
	if err != nil {
		logger.I().Error("Unable to determine system", "error", err)
		os.Exit(1)
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
