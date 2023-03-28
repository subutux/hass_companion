package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/hass/auth"
	"github.com/subutux/hass_companion/hass/mobile_app"
	"github.com/subutux/hass_companion/hass/rest"
	"github.com/subutux/hass_companion/hass/states"
	"github.com/subutux/hass_companion/hass/ws"
	"github.com/subutux/hass_companion/internal/config"
)

func main() {

	waitForClose := make(chan os.Signal, 1)
	signal.Notify(waitForClose, syscall.SIGINT, syscall.SIGTERM)
	config.Load()
	if config.Get("server") == "" {
		server := "http://192.168.0.173:8123"
		creds, err := auth.Initiate(server)
		if err != nil {
			log.Printf("Error fetching token: %s", err)
			os.Exit(1)
		}
		err = creds.Authorize()
		if err != nil {
			log.Printf("Error authorizing: %s", err)
			os.Exit(1)
		}
		config.Set("server", server)
		config.Set("auth.refreshToken", creds.RefreshToken)
		config.Set("auth.accessToken", creds.AccessToken())
		config.Set("auth.clientId", creds.ClientId)
	}

	creds := config.NewCredentialsFromConfig()
	reg := mobile_app.NewMobileAppRegistration()
	hass, err := ws.NewClient(&creds)
	if err != nil {
		log.Printf("Error creating client: %s", err)
		os.Exit(1)
	}

	StateStore := states.Store{}

	go hass.Listen()
	<-hass.Started

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
	hass.SendCommand(ws.NewSubscribeToEvents("state_changed"))

	rhass := rest.NewClient(&creds)
	registration, err := rhass.RegisterMobileApp(reg)
	config.Set("registration", *registration)
	mobile := mobile_app.NewMobileApp(registration, hass)
	mobile.EnableWebsocketPushNotifications()
	go mobile.WatchForPushNotifications(func(notification *ws.IncomingPushNotificationMessage) {
		pp.Println(notification)
	})
	go hass.MonitorConnection()

	for {
		select {
		case event := <-hass.EventChannel:
			pp.Println(event)
			ce, err := states.NewChangeEventFromIncomingEventMessage(event)
			if err == nil {
				StateStore.HandleStateChanged(ce)
			} else {
				log.Printf("Error converting event to ChangeEvent: %v", err)
			}
		case <-waitForClose:
			hass.Close()
			return
		case <-hass.PongTimeoutChannel:
			// TODO instead of closing, restart the connection
			hass.Close()
			return
		}
	}

}
