package main

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/subutux/hass_companion/internal/auth"
	"github.com/subutux/hass_companion/internal/hass"
	"github.com/subutux/hass_companion/internal/ws"
)

var connected bool
var homeAssistant *hass.Hass

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

	w.SetContent(statusStringLabel)
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	go func() {
		connected = false
		statusStringLabel.SetText("Retrieving authentication ...")
		creds, err := auth.Initiate("https://home.assistant.subutux.be")
		if err != nil {
			log.Fatal(err)
		}
		err = creds.Authorize()
		if err != nil {
			log.Fatal(err)
		}

		statusStringLabel.SetText("Connecting ...")

		homeAssistant = hass.NewHass("wss://home.assistant.subutux.be", creds)
		homeAssistant.Connect()
		for !homeAssistant.Connected {
			time.Sleep(time.Second * 1)
			log.Print("waiting for connection...")
			statusStringLabel.SetText("waiting for connection...")
		}
		statusStringLabel.SetText("Connected")
		homeAssistant.SubscribeToEventType("state_changed", func(message []byte, conn *ws.Websocket) {
			log.Printf("Received event: %s", string(message))
		})
	}()

	w.ShowAndRun()
	homeAssistant.Close()
}
