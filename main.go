package main

import (
	"log"

	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/internal/auth"
	"github.com/subutux/hass_companion/internal/hass"
)

var connected bool

func main() {
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
	err = homeAssistant.Connect()
	if err != nil {
		log.Print(err)
	}
}
