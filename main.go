package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/hass/auth"
	"github.com/subutux/hass_companion/hass/mobile_app"
	"github.com/subutux/hass_companion/hass/rest"
	"github.com/subutux/hass_companion/hass/ws"
	"github.com/subutux/hass_companion/internal/config"
)

func main() {

	waitForClose := make(chan os.Signal, 1)
	signal.Notify(waitForClose, syscall.SIGINT, syscall.SIGTERM)
	config.Load()

	url := "wss://home.assistant.subutux.be/api/websocket"
	accessToken := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiIwZTFmMDNlYTE4OWE0Yzg4YjcxZGRlNGIyYjVmNjk5OCIsImlhdCI6MTY1NzEyMzA2NywiZXhwIjoxNjU3MTI0ODY3fQ.rx6Q3OE-TPwcMo6sCWP8vYavpjBP-jes3lAdwXxEgN8"
	refreshToken := "db7f6a7ecb661a4f17d0fa383f3e5ecdfc1f82471ed8ac8bc7db6b0a1d51379532b17a486ba2a0a12c6a0a815d9d2717a4ccdf6cf891273f7305473fa2799686"
	creds := auth.NewCredentials("https://home.assistant.subutux.be", "http://localhost:9999", accessToken, refreshToken)


	reg := mobile_app.NewMobileAppRegistration()
	hass, err := ws.NewClient(url, &creds)
	if err != nil {
		log.Printf("Error creating client: %s", err)
		os.Exit(1)
	}

	hass.Listen()
	<-hass.Started

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
