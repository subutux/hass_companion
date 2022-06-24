package hass

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
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
	Socket  ws.Websocket
	RestApi *resty.Client

	Connected bool
	ticker    *time.Ticker
	lastId    int

	eventHandlers map[string][]ws.MessageHandler

	notificationId      int
	notificationHandler func(*PushNotification)
}

func NewHass(server string, credentials auth.Credentials) *Hass {
	apiUrl, _ := url.Parse(server)
	wsUrl := url.URL{
		Host: apiUrl.Host,
	}
	if apiUrl.Scheme == "https" {
		wsUrl.Scheme = "wss"
	} else {
		wsUrl.Scheme = "ws"
	}
	wsUrl.Path = "/api/websocket"
	hass := &Hass{
		Credentials:    credentials,
		Socket:         *ws.NewWebsocket(wsUrl),
		RestApi:        resty.New().SetBaseURL(apiUrl.String()),
		eventHandlers:  make(map[string][]ws.MessageHandler),
		notificationId: -1,
	}

	hass.registerRoutes()

	return hass

}

func (h *Hass) apiRequest() *resty.Request {
	return h.RestApi.R().SetAuthToken(h.Credentials.AccessToken())
}

func (h *Hass) Version() (string, error) {
	resp, err := h.apiRequest().Get("/api/")

	return resp.String(), err

}

func (h *Hass) registerRoutes() {

	h.Socket.RegisterHandler("auth_required", h.wsLogin)
	h.Socket.RegisterHandler("auth_ok", h.wsLoginSuccessfull)
	h.Socket.RegisterHandler("pong", h.pong)
	h.Socket.RegisterHandler("event", h.routeEvent)
}

func (h *Hass) Connect() {
	h.Socket.Connect()
}

func (h *Hass) Close() {
	if h.ticker != nil {
		h.ticker.Stop()
	}
	h.Socket.Destroy()
}

func (h *Hass) ping() {
	h.ticker = time.NewTicker(PING_PERIOD)
	defer h.ticker.Stop()
	for ; ; <-h.ticker.C {
		h.Socket.Send(Command{
			ID:   0,
			Type: "ping",
		})
	}
}

func (h *Hass) pong(message []byte, conn *ws.Websocket) {
	return
}

func (h *Hass) routePossiblePushNotification(id int, message []byte, conn *ws.Websocket) {
	// Notifications are not registered, bail out
	if h.notificationId == -1 {
		return
	}

	notification := PushNotification{}
	json.Unmarshal(message, &notification)
	if id == h.notificationId {
		if h.notificationHandler != nil {
			h.notificationHandler(&notification)
		}
	}

}

func (h *Hass) routeEvent(message []byte, conn *ws.Websocket) {
	var event EventMessage
	json.Unmarshal(message, &event)
	h.routePossiblePushNotification(event.ID, message, conn)
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
	h.Socket.Send(CommandSubscribeEvents{
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

func (h *Hass) NextID() int {
	h.lastId = h.lastId + 1
	return h.lastId
}

func (h *Hass) RegisterNotificationHandler(id int, f func(*PushNotification)) {
	h.notificationId = id
	h.notificationHandler = f
}
