package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/godbus/dbus/v5"

	"github.com/subutux/hass_companion/hass/auth"
	"github.com/subutux/hass_companion/hass/mobile_app"
	"github.com/subutux/hass_companion/hass/mobile_app/sensors"
	"github.com/subutux/hass_companion/hass/rest"
	"github.com/subutux/hass_companion/hass/states"
	"github.com/subutux/hass_companion/hass/ws"
	"github.com/subutux/hass_companion/internal/config"
	"github.com/subutux/hass_companion/internal/logger"
	"github.com/subutux/hass_companion/internal/themes"
	"github.com/subutux/hass_companion/internal/ui"
)

var (
	hass                  *ws.Client
	a                     fyne.App
	eventCount            int
	retries               = 0
	retryIntervalDuration = 5 * time.Second
	errorPingTimeout      = errors.New("PingTimeout")
)

func main() {
	a = app.NewWithID("be.subutux.companion")
	a.Settings().SetTheme(themes.HassLightTheme{})
	w := a.NewWindow("Companion")

	status_content := ui.NewStatusContent(&a, &w)
	content := ui.NewMainContent(&status_content)

	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("Companion",
			fyne.NewMenuItem("Status", func() {
				w.Show()
				content.Select("status")
			}))
		desk.SetSystemTrayMenu(m)
	}

	w.SetContent(content.Container())
	status_content.Status.Set(ui.StatusWaiting)
	status_content.EventsCount.Set("0")
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	waitForClose := make(chan os.Signal, 1)
	signal.Notify(waitForClose, syscall.SIGINT, syscall.SIGTERM)
	config.Load()
	status_content.SetStatus(ui.StatusConnecting)
	if config.Get("server") == "" {
		d := dialog.NewEntryDialog("Connect", "home assistant url:", func(url string) {
			config.Set("server", url)
			config.Save()
			hass = Connect(SetupAuth())
			Start(hass, &status_content, content, waitForClose)
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
		status_content.Status.Set(ui.StatusConnecting)
		hass = Connect(SetupAuth())
		Start(hass, &status_content, content, waitForClose)
	}
	w.Resize(fyne.Size{
		Width:  600,
		Height: 400,
	})
	w.ShowAndRun()
	hass.Close()
}

func Start(hass *ws.Client, status *ui.StatusContent, main *ui.MainContent, closeChannel chan os.Signal) {
	go func() {

		status.Server.Set(hass.Credentials.Server)
		go hass.Listen()
		<-hass.Started

		go hass.MonitorConnection()

		// Setup State Tracking
		//StateStore := SetupStateTracking()

		hass.SendCommand(ws.NewSubscribeToEvents("state_changed"))

		// SetupMobile
		mobile, err := SetupMobile(*hass.Credentials, main)
		if err != nil {
			logger.I().Error("Failed to setup mobile", "error", err)
			a.Quit()
		}

		status.SetStatus(ui.StatusConnected)

		for {
			select {
			case <-hass.EventChannel:
				//pp.Println(event)
				//ce, err := states.NewChangeEventFromIncomingEventMessage(event)
				eventCount += 1
				status.EventsCount.Set(strconv.Itoa(eventCount))
				// if err == nil {
				// 	// StateStore.HandleStateChanged(ce)
				// } else {
				// 	log.Printf("Error converting event to ChangeEvent: %v", err)
				// }
			case <-closeChannel:

				status.SetStatus(ui.StatusDisconnecting)
				hass.Close()
				a.Quit()
				return
			case <-hass.PongTimeoutChannel:

				status.SetStatus(ui.StatusDisconnecting, "Failed to receive a pong in time.")
				mobile.SensorCollector.Stop()
				main.ResetSensors()
				logger.I().Warn("Trying redial")
				var hassErr error
				var tries int = 1
				hassErr = hass.Redial()
				for hassErr != nil {
					logger.I().Warn("reconnecting", "error", hassErr, "try", tries)
					status.SetStatus(ui.StatusReconnecting, fmt.Sprintf("(try %v)\n%v", tries, hassErr))
					hassErr = hass.Redial()
					tries++
					if hassErr != nil {
						// back-off sleep when Redail failed
						time.Sleep(5 * time.Second)
					}
				}
				status.SetStatus(ui.StatusConnected, fmt.Sprintf("after %v tries...", tries))
				Start(hass, status, main, closeChannel)
				logger.I().Info("Restarted connection")
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
			logger.I().Error("Failed to initiate authentication", "error", err)
			os.Exit(1)
		}
		config.Save()
	} else {
		creds = config.NewCredentialsFromConfig()
	}
	err = creds.Authorize()
	if err != nil {
		logger.I().Error("Failed to authorize", "error", err)
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
		logger.I().Error("Error creating client", "error", err)
		os.Exit(1)
	}

	return hass
}

func SetupMobile(creds auth.Credentials, content *ui.MainContent) (*mobile_app.MobileApp, error) {
	rhass := rest.NewClient(&creds)
	var registration *rest.RegistrationResponse
	// Load registration from config if exists
	reg, err := config.GetStruct("registration", registration)
	if err != nil {
		// Else, register a new
		reg = mobile_app.NewMobileAppRegistration()
		registration, err = rhass.RegisterMobileApp(reg)
		if err != nil {
			return nil, err
		}
		config.Set("registration", registration)
	}

	mobile := mobile_app.NewMobileApp(registration, &creds, hass, 60*time.Second)
	//cmd := ws.NewGetWebhookCmd(registration.WebhookID, mobile_app.NewWebhookGetConfigCmd())
	// cmd := ws.NewGetConfigCmd()
	// hass.SendCommandWithCallback(cmd, func(message *ws.IncomingResultMessage) {
	// 	pp.Println(message)
	// })

	// data, err := rhass.GetConfig(registration.WebhookID)
	// pp.Println(data, err)

	// hass.SendCommandWithCallback(ws.NewGetConfigCmd(), func(message *ws.IncomingResultMessage) {
	// 	pp.Println(message)
	// })
	mobile.EnableWebsocketPushNotifications()
	go mobile.WatchForPushNotifications(mobile.FreedesktopNotifier)

	conn, err := dbus.SystemBus()
	if err == nil {
		batteries, err := sensors.DiscoverBatteries(conn)
		if err == nil {
			for _, battery := range batteries {
				mobile.SensorCollector.AddSensor(battery)
				content.AppendSensor(battery)

			}
		}
	}

	memory, err := sensors.DiscoverMemory()
	if err == nil {
		mobile.SensorCollector.AddSensor(memory)
		content.AppendSensor(memory)
	}
	load, err := sensors.DiscoverAverageLoad()
	if err == nil {
		mobile.SensorCollector.AddSensor(load)
		content.AppendSensor(load)
	}

	go mobile.SensorCollector.Collect()
	return mobile, nil
}

func SetupStateTracking() *states.Store {
	StateStore := states.Store{}
	hass.SendCommandWithCallback(ws.NewGetStatesCmd(), func(message *ws.IncomingResultMessage) {
		if !message.Success {
			logger.I().Warn("Unsuccessfull response from Home Assistant", "error", message.Error)
			return
		}
		data, err := json.Marshal(message.Result)
		if err != nil {
			return
		}
		err = json.Unmarshal(data, &StateStore.States)
		if err != nil {
			logger.I().Error("failed to decode json result to States", "error", err)
		}

	})
	return &StateStore
}
