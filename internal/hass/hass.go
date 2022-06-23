package hass

import (
	"log"
	"net/url"

	"github.com/subutux/hass_companion/internal/auth"
	"github.com/subutux/hass_companion/internal/ws"
)

type AuthMessage struct {
	Type        string `json:"type"`
	AccessToken string `json:"access_token"`
}

type Hass struct {
	auth.Credentials
	ws.Connection
	connected bool
}

func NewHass(server string, credentials auth.Credentials) *Hass {
	url, _ := url.Parse(server)
	url.Path = "/api/websocket"
	hass := &Hass{
		Credentials: credentials,
		Connection:  *ws.NewConnection(*url),
	}

	hass.registerRoutes()

	return hass

}

func (h *Hass) registerRoutes() {

	h.Connection.AddTypeHandler("auth_required", h.wsLogin)
	h.Connection.AddTypeHandler("auth_ok", h.wsLoginSuccessfull)
}

func (h *Hass) Connect() error {
	return h.Connection.Connect()
}

func (h *Hass) wsLogin(message []byte, conn *ws.Connection) {
	err := conn.Send(AuthMessage{
		Type:        "auth",
		AccessToken: h.Credentials.AccessToken(),
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (h *Hass) wsLoginSuccessfull(message []byte, conn *ws.Connection) {
	log.Printf("Connected: %s", message)
	h.connected = true
}
