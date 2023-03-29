package main

import (
	"encoding/json"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/driver/desktop"
    "fyne.io/fyne/v2/layout"
    "fyne.io/fyne/v2/widget"
    "github.com/gen2brain/beeep"
    "log"
	"os"
	"os/signal"
    "strconv"
    "syscall"

	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/hass/auth"
	"github.com/subutux/hass_companion/hass/mobile_app"
	"github.com/subutux/hass_companion/hass/rest"
	"github.com/subutux/hass_companion/hass/states"
	"github.com/subutux/hass_companion/hass/ws"
	"github.com/subutux/hass_companion/internal/config"
)

var hass *ws.Client
var eventCount int
func main() {

	a := app.New()
	w := a.NewWindow("Companion")
	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("Companion",
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
	waitForClose := make(chan os.Signal, 1)
	signal.Notify(waitForClose, syscall.SIGINT, syscall.SIGTERM)
	config.Load()
	if config.Get("server") == "" {
		d := dialog.NewEntryDialog("Connect", "home assistant url:", func(url string) {
			config.Set("server", url)
			config.Save()
			Start(statusStringLabel, Events, EventCount, w, a, waitForClose)
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
		Start(statusStringLabel, Events, EventCount, w, a, waitForClose)
	}

	w.ShowAndRun()
	hass.Close()
}

func Start(statusStringLabel *widget.Label, Events *fyne.Container, EventCount *widget.Label, w fyne.Window, a fyne.App, closeChannel chan os.Signal) {
	go func() {
		server := config.Get("server")
		var creds auth.Credentials
		var err error

		if config.Get("auth.refreshToken") == "" {
			creds, err = auth.Initiate(server)
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
		config.Set("server", server)
		config.Set("auth.refreshToken", creds.RefreshToken)
		config.Set("auth.accessToken", creds.AccessToken())
		config.Set("auth.clientId", creds.ClientId)
		config.Save()

		reg := mobile_app.NewMobileAppRegistration()
		statusStringLabel.SetText("Connecting ...")
		hass, err = ws.NewClient(&creds)
		if err != nil {
			log.Printf("Error creating client: %s", err)
			os.Exit(1)
		}

		StateStore := states.Store{}

		go hass.Listen()
		go hass.MonitorConnection()
		<-hass.Started
		statusStringLabel.SetText("Connected: Setting up States tracking...")
		hass.SendCommandWithCallback(ws.NewGetStatesCmd(), func(message *ws.IncomingResultMessage) {
			if !message.Success {
				log.Printf("home assistant responded with %s: %s", message.Error.Code, message.Error.Message)
				return
			}
			data, err := json.Marshal(message.Result)
			if err != nil {
				return
			}
			err = json.Unmarshal(data, &StateStore.States)
			if err != nil {
				log.Printf("failed to decode json result to States: %v", err)
			}

		})
		statusStringLabel.SetText("Connected: Subscribing to state changes ...")
		hass.SendCommand(ws.NewSubscribeToEvents("state_changed"))

		rhass := rest.NewClient(&creds)

		statusStringLabel.SetText("Connected: Registering as a mobile app...")
		registration, err := rhass.RegisterMobileApp(reg)
		config.Set("registration", *registration)
		mobile := mobile_app.NewMobileApp(registration, hass)

		statusStringLabel.SetText("Connected: Enabling push notifications ...")
		mobile.EnableWebsocketPushNotifications()
		go mobile.WatchForPushNotifications(func(notification *ws.IncomingPushNotificationMessage) {
			//TODO switch to https://github.com/esiqveland/notify to support actions
			beeep.Notify("Home Assistant: " + notification.Event.Title, notification.Event.Message, "icon.png")
		})

		statusStringLabel.SetText("Connected.")
		Events.Show()

		for {
			select {
			case event := <-hass.EventChannel:
				pp.Println(event)
				ce, err := states.NewChangeEventFromIncomingEventMessage(event)
				eventCount += 1
				EventCount.SetText(strconv.Itoa(eventCount))
				if err == nil {
					StateStore.HandleStateChanged(ce)
				} else {
					log.Printf("Error converting event to ChangeEvent: %v", err)
				}
				case <-closeChannel:

					statusStringLabel.SetText("disconnecting...")
					hass.Close()
					a.Quit()
					return
				case <-hass.PongTimeoutChannel:
					// TODO instead of closing, restart the connection

					statusStringLabel.SetText("Failed to receive a pong in time. Disconnecting ...")
					hass.Close()
					a.Quit()
					return
			}
		}
	}()
}
