package mobile_app

import (
    "errors"
    "fmt"
    "github.com/godbus/dbus/v5"
)

const (

	dbusNotificationsInterface = "org.freedesktop.Notifications"  // DBUS Interface
	dbusObjectPath             = "/org/freedesktop/Notifications" // the DBUS object path
	callGetServerInformation   = "org.freedesktop.Notifications.GetServerInformation"
)

// ServerInformation is a holder for information returned by
// GetServerInformation call.
type ServerInformation struct {
	Name        string
	Vendor      string
	Version     string
	SpecVersion string
}

func GetServerInformation() (*ServerInformation, error) {
	conn, err := dbus.SessionBus()
		if err != nil {
		return nil, fmt.Errorf("error connecting to SessionBus: %v", err)
		}
	obj := conn.Object(dbusNotificationsInterface, dbusObjectPath)
	if obj == nil {
		return nil, errors.New("error creating dbus call object")
	}
	call := obj.Call(callGetServerInformation, 0)
	if call.Err != nil {
		return nil, fmt.Errorf("error calling %v: %v", callGetServerInformation, call.Err)
	}

	ret := ServerInformation{}
	err = call.Store(&ret.Name, &ret.Vendor, &ret.Version, &ret.SpecVersion)
	if err != nil {
		return nil, fmt.Errorf("error reading %v return values: %v", callGetServerInformation, err)
	}
	return &ret, nil
}