package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/internal/auth"
	"github.com/subutux/hass_companion/internal/hass"
	"github.com/subutux/hass_companion/internal/ws"
)

var connected bool

func main() {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	connected = false
	creds, err := auth.Initiate("https://home.assistant.subutux.be")
	if err != nil {
		log.Fatal(err)
	}
	err = creds.Authorize()
	if err != nil {
		log.Fatal(err)
	}

	pp.Println(creds)

	homeAssistant := hass.NewHass("wss://home.assistant.subutux.be", creds)
	homeAssistant.Connect()
	for !homeAssistant.Connected {
		time.Sleep(time.Second * 1)
		log.Print("waiting for connection...")
	}
	homeAssistant.SubscribeToEventType("state_changed", func(message []byte, conn *ws.Websocket) {
		log.Printf("Received event: %s", string(message))
	})

	for {
		select {
		case <-interrupt:
			log.Println("interrupt")
			homeAssistant.Close()
			return
		}
	}
}
