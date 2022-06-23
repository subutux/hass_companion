package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/subutux/hass_companion/internal/ws/messages"
)

type MessageHandler func(message []byte, conn *Connection)

type Connection struct {
	Endpoint url.URL
	ws       *websocket.Conn
	done     chan struct{}
	out      chan []byte
	handlers map[string]MessageHandler
	lastId   int
}

func NewConnection(endpoint url.URL) *Connection {
	return &Connection{
		Endpoint: endpoint,
		handlers: make(map[string]MessageHandler),
	}
}

func (c *Connection) Connect() (err error) {
	// https://tradermade.com/tutorials/golang-websocket-client/
	var resp *http.Response
	c.ws, resp, err = websocket.DefaultDialer.Dial(c.Endpoint.String(), nil)
	log.Printf("Connecting to %s", c.Endpoint.String())
	if err != nil {
		log.Printf("handshake failed with status %d", resp.StatusCode)
		return err
	}
	c.done = make(chan struct{})
	ticker := time.NewTicker(time.Second * 60)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	go func() {
		defer close(c.done)
		for {
			_, message, err := c.ws.ReadMessage()
			if err != nil {
				log.Printf("error reading message: %s", err)
				return
			}

			log.Printf("received: %s", message)
			c.route(message)
		}
	}()

	for {
		log.Println("for")
		select {
		case <-c.done:
			return
		case msg := <-c.out:
			log.Printf("sending: %s", msg)
			err = c.ws.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("error writing message: %s", err)
				return
			}

			log.Printf("sent: %s", msg)
		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(time.Second * 10))
			if err = c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping: %s", err)
				return
			}
		}
	}
}

func (c *Connection) AddTypeHandler(Type string, handler MessageHandler) {
	c.handlers[Type] = handler
	log.Printf("registered handler for %s", Type)
}

func (c *Connection) Send(message interface{}) error {
	data, err := json.Marshal(message)

	log.Printf("Sending %s", string(data))
	if err != nil {
		return err
	}

	c.out <- data
	return nil

}

func (c *Connection) route(message []byte) {
	var msg messages.Message
	err := json.Unmarshal(message, &msg)
	if err != nil {
		log.Printf("Error decoding message: %s", err)
	}

	log.Print(c.handlers)

	for k, mh := range c.handlers {
		log.Printf("k = %s; msg.Type = %s", k, msg.Type)
		if msg.Type == k {
			go mh(message, c)
			return
		}
	}
	log.Printf("unhandled message of type %s", msg.Type)
}
