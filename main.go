package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/godbus/dbus/v5"

	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/hass/auth"
	"github.com/subutux/hass_companion/hass/mobile_app"
	"github.com/subutux/hass_companion/hass/mobile_app/sensors"
	"github.com/subutux/hass_companion/hass/rest"
	"github.com/subutux/hass_companion/hass/states"
	"github.com/subutux/hass_companion/hass/ws"
	"github.com/subutux/hass_companion/internal/config"
)

var (
	hass                  *ws.Client
	eventCount            int
	retries               = 0
	retryIntervalDuration = 5 * time.Second
	errorPingTimeout      = errors.New("PingTimeout")
)

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
			Start(statusStringLabel, Events, EventCount, a, waitForClose)
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

		statusStringLabel.SetText("Connecting ...")
		Start(statusStringLabel, Events, EventCount, a, waitForClose)
	}

	w.ShowAndRun()
	hass.Close()
}

func Start(statusStringLabel *widget.Label, Events *fyne.Container, EventCount *widget.Label, a fyne.App, closeChannel chan os.Signal) {
	go func() {

		creds := SetupAuth()

		hass = Connect(creds)

		go hass.MonitorConnection()

		// Setup State Tracking
		//StateStore := SetupStateTracking()

		hass.SendCommand(ws.NewSubscribeToEvents("state_changed"))

		// SetupMobile
		mobile := SetupMobile(creds)
		statusStringLabel.SetText("Connected.")
		Events.Show()

		for {
			select {
			case <-hass.EventChannel:
				//pp.Println(event)
				//ce, err := states.NewChangeEventFromIncomingEventMessage(event)
				eventCount += 1
				EventCount.SetText(strconv.Itoa(eventCount))
				// if err == nil {
				// 	// StateStore.HandleStateChanged(ce)
				// } else {
				// 	log.Printf("Error converting event to ChangeEvent: %v", err)
				// }
			case <-closeChannel:

				statusStringLabel.SetText("disconnecting...")

				hass.Close()
				a.Quit()
				return
			case <-hass.PongTimeoutChannel:
				// TODO instead of closing, restart the connection
				statusStringLabel.SetText("Failed to receive a pong in time. Disconnecting ...")
				// Test redial
				mobile.SensorCollector.Stop()
				log.Print("Trying redial")
				var hassErr error
				var tries int = 0
				for hassErr != nil {
					time.Sleep(5 * time.Second)
					log.Printf("%v:  reconnecting (try %v)", hassErr, tries)
					hassErr = hass.Redial()
					tries++
				}

				statusStringLabel.SetText(fmt.Sprintf("connected after %v tries...", tries))
				go hass.Listen()

				go hass.MonitorConnection()

				SetupMobile(creds)
				log.Println("Restarted connection")
				return
			}
		}
	}()
}

func SetupAuth() auth.Credentials {
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
	return creds
}

func Connect(creds auth.Credentials) *ws.Client {
	hass, err := ws.NewClient(&creds)
	if err != nil {
		log.Printf("Error creating client: %s", err)
		os.Exit(1)
	}

	go hass.Listen()
	<-hass.Started

	return hass
}

func SetupMobile(creds auth.Credentials) *mobile_app.MobileApp {
	rhass := rest.NewClient(&creds)

	// TODO Load registration from config if exists
	reg := mobile_app.NewMobileAppRegistration()
	registration, err := rhass.RegisterMobileApp(reg)
	config.Set("registration", *registration)

	mobile := mobile_app.NewMobileApp(registration, &creds, hass, 60*time.Second)
	hass.SendCommandWithCallback(ws.NewGetWebhookCmd(registration.WebhookID, mobile_app.NewWebhookGetConfigCmd()), func(message *ws.IncomingResultMessage) {
		pp.Println(message)
	})
	mobile.EnableWebsocketPushNotifications()
	// go mobile.WatchForPushNotifications(func(notification *ws.IncomingPushNotificationMessage) {
	// 	//TODO switch to https://github.com/esiqveland/notify to support actions

	// 	beeep.Notify("Home Assistant: "+notification.Event.Title, notification.Event.Message, "icon.png")
	// })

	go mobile.WatchForPushNotifications(mobile.FreedesktopNotifier)

	conn, err := dbus.SystemBus()
	if err == nil {
		batteries, err := sensors.DiscoverBatteries(conn)
		if err == nil {
			for _, battery := range batteries {
				mobile.SensorCollector.AddSensor(battery)
			}
		}
	}

	memory, err := sensors.DiscoverMemory()
	if err == nil {
		mobile.SensorCollector.AddSensor(memory)
	}
	load, err := sensors.DiscoverAverageLoad()
	if err == nil {
		mobile.SensorCollector.AddSensor(load)
	}

	go mobile.SensorCollector.Collect()
	return mobile
}

func SetupStateTracking() *states.Store {
	StateStore := states.Store{}
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
	return &StateStore
}
