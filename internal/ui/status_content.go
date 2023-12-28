package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/subutux/hass_companion/internal/icons"
)

const (
	StatusConnected     string = "Connected"
	StatusConnecting    string = "Connecting"
	StatusWaiting              = "Waiting"
	StatusDisconnecting        = "Disconnecting"
	StatusDisconnected         = "Disconnected"
	StatusReconnecting         = "Reconnecting"
)

type StatusContent struct {
	Title       string
	EventsCount binding.String
	Server      binding.String
	Status      binding.String
	app         *fyne.App
	window      *fyne.Window
	logo        *canvas.Image
}

func NewStatusContent(app *fyne.App, window *fyne.Window) StatusContent {
	m := StatusContent{
		Title:       "Status",
		EventsCount: binding.NewString(),
		Server:      binding.NewString(),
		Status:      binding.NewString(),
		app:         app,
		window:      window,
	}

	m.EventsCount.Set("0")
	m.Server.Set("")
	m.Status.Set(StatusWaiting)

	return m

}

func (m *StatusContent) SetStatus(status ...string) {
	switch status[0] {
	case StatusConnected:
		m.setIcon(icons.Default)
		break
	case StatusConnecting, StatusDisconnecting, StatusWaiting:
		m.setIcon(icons.Waiting)
		break
	case StatusDisconnected, StatusReconnecting:
		m.setIcon(icons.Disconnected)
	}

	m.Status.Set(strings.Join(status, " "))
}

func (m *StatusContent) setIcon(icon fyne.Resource) {
	app := *m.app
	window := *m.window
	if desk, ok := app.(desktop.App); ok {
		desk.SetSystemTrayIcon(icon)
		window.SetIcon(icon)
		m.logo.Resource = icon
		m.logo.Refresh()
	}
}

func (m *StatusContent) Container() fyne.CanvasObject {
	m.logo = canvas.NewImageFromResource(icons.Default)
	m.logo.SetMinSize(fyne.Size{
		Width:  100,
		Height: 100,
	})
	ServerLabel := widget.NewLabelWithData(m.Server)
	ServerLabel.TextStyle.Bold = true
	StatusLabel := widget.NewLabelWithData(m.Status)
	EventTitle := widget.NewLabelWithStyle("Events received",
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		})
	EventsCount := widget.NewLabelWithData(m.EventsCount)
	Status := container.NewHBox(ServerLabel, StatusLabel)
	Events := container.NewHBox(EventTitle, EventsCount)
	return container.NewCenter(
		container.NewVBox(container.NewCenter(m.logo),
			container.NewCenter(
				container.NewVBox(Status, Events),
			),
		),
	)

}
