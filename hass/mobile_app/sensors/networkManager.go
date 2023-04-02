package sensors

import "github.com/godbus/dbus/v5"

type NetworkInterface struct {
	Sensor
}

type ActiveConnection struct {
	Name  string
	SSID  string
	Type  string
	BSSID string
	tx    uint64
	rx    uint64
}

func getActiveConnection(conn *dbus.Conn) *ActiveConnection {
	activeConnections, err := conn.Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager").
		GetProperty("org.freedesktop.NetworkManager.ActiveConnections")
	if err != nil {
		return nil
	}

	for _, path := range activeConnections.Value().([]dbus.ObjectPath) {

		o := conn.Object("org.freedesktop.NetworkManager", path)
		vpn, _ := o.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Vpn")
		isDefault, _ := o.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Default")
		if !vpn.Value().(bool) && isDefault.Value().(bool) {

			name, _ := o.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Id")
			t, _ := o.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Type")
			// devices, err := o.GetProperty("org.freedesktop.NetworkManager.Connection.Active.Devices")
			ob, _ := o.GetProperty("org.freedesktop.NetworkManager.Connection.Active.SpecificObject")

			connection := ActiveConnection{
				Name: name.Value().(string),
				Type: t.Value().(string),
			}
			if t.Value().(string) == "802-11-wireless" {
				ap := getAccessPoint(conn, ob.Value().(dbus.ObjectPath))
				connection.SSID = ap.SSID
				connection.BSSID = ap.BSSID
			}

			return &connection
		}

	}
	return nil
}

type AP struct {
	BSSID string
	SSID  string
}

func getAccessPoint(conn *dbus.Conn, op dbus.ObjectPath) (ap *AP) {

	o := conn.Object("org.freedesktop.NetworkManager", op)
	ssid, err := o.GetProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid")
	if err != nil {
		return ap
	}
	hwaddress, err := o.GetProperty("org.freedesktop.NetworkManager.AccessPoint.HwAddress")
	if err != nil {
		return ap
	}

	ap.SSID = ssid.Value().(string)
	ap.BSSID = hwaddress.Value().(string)

	return ap
}

// TODO
