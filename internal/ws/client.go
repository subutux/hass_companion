package ws

import (
	"encoding/json"
	"log"
	"net/url"

	"github.com/sacOO7/gowebsocket"
	"github.com/subutux/hass_companion/internal/ws/messages"
)

type MessageHandler func(message []byte, conn *Websocket)

type Websocket struct {
	Endpoint url.URL
	socket   gowebsocket.Socket

	handlers map[string]MessageHandler
}

func NewWebsocket(endpoint url.URL) *Websocket {
	return &Websocket{
		Endpoint: endpoint,
		handlers: make(map[string]MessageHandler),
	}
}

func (w *Websocket) Destroy() {
	w.socket.Close()
}

func (w *Websocket) Connect() {
	w.socket = gowebsocket.New(w.Endpoint.String())
	w.socket.OnConnected = func(socket gowebsocket.Socket) {
		log.Printf("Connected to websocket %s", w.Endpoint.String())
	}
	w.socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Printf("Connection failed to websocket %s: %s", w.Endpoint.String(), err)
		w.Destroy()
	}
	w.socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		log.Printf("Received: %s", message)
		w.route([]byte(message))
	}
	w.socket.Connect()
}

func (w *Websocket) RegisterHandler(on string, f MessageHandler) {
	w.handlers[on] = f
}

func (w *Websocket) Send(v interface{}) error {
	data, err := json.Marshal(v)

	log.Printf("Sending %s", string(data))
	if err != nil {
		return err
	}
	w.socket.SendText(string(data))
	return nil
}

func (w *Websocket) route(message []byte) {
	var msg messages.Message
	err := json.Unmarshal(message, &msg)
	if err != nil {
		log.Printf("Error decoding message %s: %s", message, err)
		return
	}

	for k, mh := range w.handlers {
		if msg.Type == k {
			go func() {
				mh(message, w)
			}()
			return
		}
	}
	log.Printf("unhandled message of type %s", msg.Type)
}
