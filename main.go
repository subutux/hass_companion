package main

import (
	"log"
	"net/url"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/gen2brain/beeep"
	"github.com/subutux/hass_companion/internal/auth"
	"github.com/subutux/hass_companion/internal/config"
	"github.com/subutux/hass_companion/internal/hass"
	"github.com/subutux/hass_companion/internal/ws"
)

var homeAssistant *hass.Hass
var eventCount int

func main() {

	a := app.New()
	w := a.NewWindow("Status")

	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("MyApp",
			fyne.NewMenuItem("Status", func() {
				w.Show()
			}))
		desk.SetSystemTrayMenu(m)
	}

	statusStringLabel := widget.NewLabel("loading")
	EventTitle := widget.NewLabel("Events received")
	EventTitle.TextStyle = fyne.TextStyle{
		Bold: true,
	}
	EventCount := widget.NewLabel("0")
	Events := container.New(layout.NewHBoxLayout(), EventTitle, EventCount)
	Events.Hide()
	w.SetContent(container.New(layout.NewVBoxLayout(), statusStringLabel, Events))
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	config.Load()
	if config.Get("server") == "" {
		d := dialog.NewEntryDialog("Connect", "home assistant url:", func(url string) {
			config.Set("server", url)
			config.Save()
			Start(statusStringLabel, Events, EventCount, w)
		}, w)
		w.Resize(fyne.Size{
			Width:  600,
			Height: 400,
		})
		d.Resize(fyne.Size{
			Width:  500,
			Height: 300,
		})
		d.Show()
		w.RequestFocus()
	} else {

		Start(statusStringLabel, Events, EventCount, w)
	}
	w.ShowAndRun()
	homeAssistant.Close()
}

func Start(statusStringLabel *widget.Label, Events *fyne.Container, EventCount *widget.Label, w fyne.Window) {
	go func() {
		server, err := url.Parse(config.Get("server"))
		var creds auth.Credentials
		statusStringLabel.SetText("Retrieving authentication ...")
		if config.Get("auth.refreshToken") == "" {
			creds, err = auth.Initiate(server.String())
			if err != nil {
				log.Fatal(err)
			}
			config.Save()
		} else {
			creds = config.NewCredentialsFromConfig()
		}

		err = creds.Authorize()
		if err != nil {
			log.Fatal(err)
		}

		config.Set("auth.refreshToken", creds.RefreshToken)
		config.Set("auth.accessToken", creds.AccessToken())
		config.Set("auth.clientId", creds.ClientId)
		config.Save()

		statusStringLabel.SetText("Connecting ...")
		homeAssistant = hass.NewHass(server.String(), creds)
		homeAssistant.Connect()
		for !homeAssistant.Connected {
			time.Sleep(time.Second * 1)
			log.Print("waiting for connection...")
			statusStringLabel.SetText("waiting for connection...")
		}
		statusStringLabel.SetText("Connected")
		Events.Show()
		go func() {
			time.Sleep(time.Second * 1)
			w.Hide()
		}()
		log.Print(homeAssistant.Version())
		homeAssistant.SubscribeToEventType("state_changed", func(message []byte, conn *ws.Websocket) {
			log.Printf("Received event: %s", string(message))
			eventCount = eventCount + 1
			EventCount.SetText(strconv.Itoa(eventCount))
		})

		device := homeAssistant.RegisterCompanion()
		device.SetupPush(func(notification *hass.PushNotification) {
			beeep.Notify(notification.Event.Title, notification.Event.Message, "")
		})
		// pp.Println(mobile_device)
	}()
}
