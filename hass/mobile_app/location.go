package mobile_app

import (
	"errors"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/godbus/dbus/v5"
	"github.com/subutux/hass_companion/internal/logger"
)

func (m *MobileApp) SendLocationUpdate(location *Location) error {
	update_location := NewWebhookUpdateLocationCmd(location)
	webhook, err := m.WebhookUrl()
	if err != nil {
		return err
	}
	r, err := resty.New().R().
		SetBody(update_location).
		Post(webhook)
	if err != nil {
		return err
	}

	if r.IsError() {
		return fmt.Errorf("Error sending location update: %v", r.Error())
	}
	logger.I().Info("UpdateLocationResponse", "webhook", webhook, "post", update_location, "body", string(r.Body()))
	return nil
}

func (m *MobileApp) MonitorLocation() error {

	geo, err := NewGeoClue()
	if err != nil {
		return err
	}

	path := geo.CreateGeoClient()
	err = geo.RegisterSignal(path)
	if err != nil {
		return err
	}

	logger.I().Info("Registered GeoClue2 Client", "path", path)

	loc, err := geo.Location(path)
	if err != nil {
		logger.I().Info("Initial location failed", "err", err)

	} else {
		m.SendLocationUpdate(loc)
		logger.I().Info("Initial location", "location", loc)
	}

	go geo.Watch(path)
	for l := range geo.LocationChan {
		logger.I().Info("Received location update", "location", l)
		m.SendLocationUpdate(&l)
	}

	return nil
}

type Location struct {
	Latitude    float64
	Longitude   float64
	Accuracy    float64
	Altitude    float64
	Speed       float64
	Heading     float64
	Description string
	// Timestamp time.Time
}

type GeoClue struct {
	SystemBus    *dbus.Conn
	Manager      dbus.BusObject
	updateChan   chan *dbus.Signal
	LocationChan chan Location
}

func NewGeoClue() (*GeoClue, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	obj := conn.Object("org.freedesktop.GeoClue2",
		dbus.ObjectPath("/org/freedesktop/GeoClue2/Manager"))

	return &GeoClue{
		SystemBus:    conn,
		Manager:      obj,
		LocationChan: make(chan Location, 10),
	}, nil

}

func (g *GeoClue) CreateGeoClient() dbus.ObjectPath {

	call := g.Manager.Call("org.freedesktop.GeoClue2.Manager.CreateClient", 0)

	if call.Err != nil {
		logger.I().Error("Error fetching GeoClue2 Client", "error", call.Err)
		return "/"
	}

	logger.I().Info("GeoClue2 Client", "call", call)

	return call.Body[0].(dbus.ObjectPath)

}

func (g *GeoClue) DeleteGeoClient(clientPath dbus.ObjectPath) {

	call := g.Manager.Call("org.freedesktop.GeoClue2.Manager.DeleteClient", 0, clientPath)

	if call.Err != nil {
		logger.I().Error("Error fetching GeoClue2 Client", "error", call.Err)
	}
	return

}

func (g *GeoClue) RegisterSignal(clientPath dbus.ObjectPath) error {
	var err error
	client := g.SystemBus.Object("org.freedesktop.GeoClue2", clientPath)

	logger.I().Info("register client", "path", clientPath)
	v, err := client.GetProperty("org.freedesktop.GeoClue2.Client.DesktopId")

	logger.I().Info("deskid", "id", v, "err", err)
	// BUG: No such interface “org.freedesktop.DBus.Properties” on object at path /org/freedesktop/GeoClue2/Client/21
	var desktop dbus.Variant = dbus.MakeVariant("hass")
	call := client.Call("org.freedesktop.DBus.Properties.Set", 0, "org.freedesktop.GeoClue2.Client", "DesktopId", desktop)

	if call.Err != nil {
		g.DeleteGeoClient(clientPath)
		return fmt.Errorf("Failed to set DesktopId %v", call.Err)
	}
	call = client.Call("org.freedesktop.DBus.Properties.Set", 0, "org.freedesktop.GeoClue2.Client", "RequestedAccuracyLevel", dbus.MakeVariant(uint(8)))

	if call.Err != nil {
		g.DeleteGeoClient(clientPath)
		return fmt.Errorf("Failed to set Accuracy to 8: %v", call.Err)
	}

	call = client.Call("org.freedesktop.GeoClue2.Client.Start", 0)
	if call.Err != nil {
		g.DeleteGeoClient(clientPath)
		return call.Err
	}

	logger.I().Info("Started client", "call", call)
	g.SystemBus.AddMatchSignal(dbus.WithMatchObjectPath(clientPath))
	g.updateChan = make(chan *dbus.Signal, 1)
	g.SystemBus.Signal(g.updateChan)

	return nil
}

func (g *GeoClue) Watch(clientPath dbus.ObjectPath) {
	client := g.SystemBus.Object("org.freedesktop.GeoClue2.Client", clientPath)
	defer client.Call("org.freedesktop.GeoClue2.Client.Stop", 0)
	defer g.SystemBus.RemoveSignal(g.updateChan)
	for v := range g.updateChan {
		logger.I().Info("Received Geo Signal", "name", v.Name)
		if v.Name == "org.freedesktop.GeoClue2.Client.LocationUpdated" {
			old := v.Body[0].(dbus.ObjectPath)
			new := v.Body[1].(dbus.ObjectPath)
			loc, err := g.HandleUpdate(old, new)

			if err != nil {
				logger.I().Error("Failed to handle update", "error", err)
			} else {
				g.LocationChan <- *loc
			}
		}
	}

}

func (g *GeoClue) Location(clientPath dbus.ObjectPath) (*Location, error) {

	client := g.SystemBus.Object("org.freedesktop.GeoClue2", clientPath)
	lpv, err := client.GetProperty("org.freedesktop.GeoClue2.Client.Location")
	if err != nil {
		return nil, err
	}
	loc := lpv.Value().(dbus.ObjectPath)
	if loc == "/" {
		return nil, errors.New("No location available")
	}
	return g.HandleUpdate(loc, loc)

}

func (g *GeoClue) HandleUpdate(old, new dbus.ObjectPath) (*Location, error) {

	client := g.SystemBus.Object("org.freedesktop.GeoClue2", new)
	latv, err := client.GetProperty("org.freedesktop.GeoClue2.Location.Latitude")
	if err != nil {
		return nil, err
	}
	lonv, err := client.GetProperty("org.freedesktop.GeoClue2.Location.Longitude")
	if err != nil {
		return nil, err
	}

	accv, err := client.GetProperty("org.freedesktop.GeoClue2.Location.Accuracy")
	if err != nil {
		return nil, err
	}
	altv, err := client.GetProperty("org.freedesktop.GeoClue2.Location.Altitude")
	if err != nil {
		return nil, err
	}
	spv, err := client.GetProperty("org.freedesktop.GeoClue2.Location.Speed")
	if err != nil {
		return nil, err
	}
	heav, err := client.GetProperty("org.freedesktop.GeoClue2.Location.Heading")
	if err != nil {
		return nil, err
	}
	dscv, err := client.GetProperty("org.freedesktop.GeoClue2.Location.Description")
	if err != nil {
		return nil, err
	}

	// tmv, err := client.GetProperty("org.freedesktop.GeoClue2.Location.Timestamp")
	// if err != nil {
	// 	return nil, err
	// }

	return &Location{
		Latitude:    latv.Value().(float64),
		Longitude:   lonv.Value().(float64),
		Accuracy:    accv.Value().(float64),
		Altitude:    altv.Value().(float64),
		Speed:       spv.Value().(float64),
		Heading:     heav.Value().(float64),
		Description: dscv.Value().(string),
		// Timestamp:   tmv.Value().(time.Time),
	}, nil
}
