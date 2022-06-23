package hass

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/subutux/hass_companion/internal/auth"
	"github.com/subutux/hass_companion/internal/ws"
)

const (
	PING_PERIOD = 30 * time.Second
)

type AuthMessage struct {
	Type        string `json:"type"`
	AccessToken string `json:"access_token"`
}

type Command struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

type CommandSubscribeEvents struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	EventType string `json:"event_type"`
}

type ServerResponseMessage struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result"`
}
type Hass struct {
	auth.Credentials
	Connection    ws.Websocket
	Connected     bool
	ticker        *time.Ticker
	lastId        int
	eventHandlers map[string][]ws.MessageHandler
}

func NewHass(server string, credentials auth.Credentials) *Hass {
	url, _ := url.Parse(server)
	url.Path = "/api/websocket"
	hass := &Hass{
		Credentials:   credentials,
		Connection:    *ws.NewWebsocket(*url),
		eventHandlers: make(map[string][]ws.MessageHandler),
	}

	hass.registerRoutes()

	return hass

}

func (h *Hass) registerRoutes() {

	h.Connection.RegisterHandler("auth_required", h.wsLogin)
	h.Connection.RegisterHandler("auth_ok", h.wsLoginSuccessfull)
	h.Connection.RegisterHandler("pong", h.pong)
	h.Connection.RegisterHandler("event", h.routeEvent)
}

func (h *Hass) Connect() {
	h.Connection.Connect()
}

func (h *Hass) Close() {
	if h.ticker != nil {
		h.ticker.Stop()
	}
	h.Connection.Destroy()
}

func (h *Hass) ping() {
	h.ticker = time.NewTicker(PING_PERIOD)
	defer h.ticker.Stop()
	for ; ; <-h.ticker.C {
		h.Connection.Send(Command{
			ID:   0,
			Type: "ping",
		})
	}
}

func (h *Hass) pong(message []byte, conn *ws.Websocket) {
	return
}

func (h *Hass) routeEvent(message []byte, conn *ws.Websocket) {
	var event EventMessage
	json.Unmarshal(message, &event)
	if handlers, ok := h.eventHandlers[event.Event.EventType]; ok {
		for _, mh := range handlers {
			mh(message, conn)
		}
	}
}

func (h *Hass) wsLogin(message []byte, conn *ws.Websocket) {
	err := conn.Send(AuthMessage{
		Type:        "auth",
		AccessToken: h.Credentials.AccessToken(),
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (h *Hass) wsLoginSuccessfull(message []byte, conn *ws.Websocket) {
	log.Printf("Connected to %s!", h.Server)
	h.Connected = true
}

func (h *Hass) SubscribeToEventType(eventType string, f ws.MessageHandler) {
	id := h.lastId + 1
	h.Connection.Send(CommandSubscribeEvents{
		ID:        id,
		Type:      "subscribe_events",
		EventType: eventType,
	})

	h.registerEventHandler(eventType, f)

	h.lastId = id
}

func (h *Hass) registerEventHandler(eventType string, f ws.MessageHandler) {
	if _, ok := h.eventHandlers[eventType]; !ok {
		h.eventHandlers[eventType] = []ws.MessageHandler{}
	}

	h.eventHandlers[eventType] = append(h.eventHandlers[eventType], f)
}
