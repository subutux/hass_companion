package ws

import (
	"bytes"
	"errors"
    "fmt"
    "log"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/hass/auth"
)

const avgReadMsgSizeBytes = 1024
var NotAuthenticatedError error = errors.New("not authenticated")
type Client struct {
	*auth.Credentials
	Authenticated bool
	Ready         bool
	Started       chan struct{}
	Conn          *websocket.Conn
	Sequence      int64

	EventChannel            chan *IncomingEventMessage
	PushNotificationChannel chan *IncomingPushNotificationMessage
	ResultChannel           chan *IncomingResultMessage

	writeChan chan interface{}

	PingIntervalTimer  *time.Ticker
	PongChannel        chan *IncomingPongMessage
	PongTimeoutChannel chan bool
	quitPingWatchdog   chan struct{}

	quitWriterChan chan struct{}
	resetTimerChan chan struct{}
	closed         int32

	callbacks map[int64]func(message *IncomingResultMessage)

	ListenError ClientError
}

// detectWebsocketUrl tries to determine the websocket url from the server url
func detectWebsocketUrl(server string) (wsUrl url.URL, err error) {
	serverUrl, err := url.Parse(server)
	if err != nil {
		return wsUrl, err
	}
	wsUrl.Scheme = "ws"
	if serverUrl.Scheme == "https" {
		wsUrl.Scheme = "wss"
	}

	wsUrl.Host = serverUrl.Host
	wsUrl.Path = "/api/websocket"
	pp.Print(server, wsUrl)
	return wsUrl, nil

}

func NewClient(credentials *auth.Credentials) (*Client, error) {
	server, err := detectWebsocketUrl(credentials.Server)
	if err != nil {
		return nil, err
	}
	dailer := websocket.DefaultDialer
	dailer.HandshakeTimeout = 5 * time.Second
	conn, _, err := dailer.Dial(server.String(), nil)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Credentials:   credentials,
		Authenticated: false,
		Conn:          conn,
		Sequence:      1,

		Ready:   false,
		Started: make(chan struct{}),

		EventChannel:            make(chan *IncomingEventMessage, 1000),
		ResultChannel:           make(chan *IncomingResultMessage, 1000),
		PushNotificationChannel: make(chan *IncomingPushNotificationMessage, 1000),
		writeChan:               make(chan interface{}),

		PongChannel:        make(chan *IncomingPongMessage),
		PingIntervalTimer:  time.NewTicker(5 * time.Second),
		PongTimeoutChannel: make(chan bool, 1),

		quitPingWatchdog: make(chan struct{}),
		quitWriterChan:   make(chan struct{}),
		resetTimerChan:   make(chan struct{}),

		callbacks: make(map[int64]func(message *IncomingResultMessage)),

		closed: 0,
	}

	go client.writer()
	return client, nil
}

// Listen starts the read loop of the websocket client.
func (c *Client) Listen() {
	// This loop can exit in 2 conditions:
	// 1. Either the connection breaks naturally.
	// 2. Close was explicitly called, which closes the connection manually.
	//
	// Due to the way the API is written, there is a requirement that a client may NOT
	// call Listen at all and can still call Close and Connect.
	// Therefore, we let the cleanup of the reader stuff rely on closing the connection
	// and then we do the cleanup in the defer block.
	//
	// First, we close some channels and then CAS to 1 and proceed to close the writer chan also.
	// This is needed because then the defer clause does not double-close the writer when (2) happens.
	// But if (1) happens, we set the closed bit, and close the rest of the stuff.
	defer func() {
		close(c.EventChannel)
		close(c.ResultChannel)
		close(c.PushNotificationChannel)
		close(c.quitPingWatchdog)
		close(c.PongTimeoutChannel)
		close(c.resetTimerChan)
		if !c.Ready {
			close(c.Started)
		}
		c.Close()
	}()

	var buf bytes.Buffer
	buf.Grow(avgReadMsgSizeBytes)

	for {
		// Reset buffer.
		buf.Reset()
		_, r, err := c.Conn.NextReader()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				c.ListenError = NewClientError("Client.Listen", err)
			}
			return
		}
		// Use pre-allocated buffer.
		_, err = buf.ReadFrom(r)
		if err != nil {
			c.ListenError = NewClientError("Client.Listen", err)
			return
		}

		msg, jsonErr := IncomingMessageFromJSON(bytes.NewReader(buf.Bytes()))
		if jsonErr != nil {
			log.Printf("Failed to decode from json: %s", jsonErr)
			continue
		}

		if msg.Is("event") {

			// First, try to decode it as an push notification
			notification, jsonErr := IncomingPushNotificationMessageFromJSON(bytes.NewReader(buf.Bytes()))
			if jsonErr != nil {
				log.Printf("Failed to decode event from json: %s", jsonErr)
			} else if notification.Event.Message != "" {
				log.Printf("received push notification %v", buf.String())

				c.PushNotificationChannel <- notification
				continue
			}
			// then continue to event processing
			event, jsonErr := IncomingEventMessageFromJSON(bytes.NewReader(buf.Bytes()))
			if jsonErr != nil {
				log.Printf("Failed to decode event from json: %s", jsonErr)
			} else {
				log.Printf("received %s event", event.Event.EventType)
				c.EventChannel <- event
				continue
			}
		}

		if msg.Is("result") {
			result, jsonErr := IncomingResultMessageFromJSON(bytes.NewReader(buf.Bytes()))
			if jsonErr != nil {
				log.Printf("Failed to decode result from json: %s", jsonErr)
			} else {
				log.Printf("received result with error %v and success %v", result.Error, result.Success)

				// first check if we have a callback set for this id
				cb, ok := c.callbacks[msg.ID]
				if ok {
					log.Printf("Calling callback for id %v", msg.ID)
					cb(result)
					// Delete the callback afterwards
					delete(c.callbacks, msg.ID)
					continue
				}
				c.ResultChannel <- result
			}
			continue
		}

		if msg.In("auth_required", "auth_ok", "auth_invalid") {
			result, jsonErr := IncomingMessageFromJSON(bytes.NewReader(buf.Bytes()))
			if jsonErr != nil {
				log.Printf("Failed to decode result from json: %s", jsonErr)
			} else {
				c.authenticate(result)
			}
			continue
		}

		if msg.Is("pong") {
			result, jsonErr := IncomingPongMessageFromJSON(bytes.NewReader(buf.Bytes()))
			if jsonErr != nil {
				log.Printf("Failed to decode result from json: %s", jsonErr)
			} else {
				c.PongChannel <- result
			}
			continue
		}
	}
}

// MonitorConnection periodically sends pings over the websocket connection
// to home assistant and expects a pong message back within one second.
// If we did not received a pong in time, a bool will be posted to the Client.PongTimeoutChannel
// to indicate that there is a problem with the connection.
func (c *Client) MonitorConnection() {
	// Periodically send a Ping
	for {
		select {
		case <-c.quitPingWatchdog:
			c.PingIntervalTimer.Stop()
			return
		case t := <-c.PingIntervalTimer.C:
			log.Printf("Ping at %v", t)
			err := c.SendCommand(NewPingCmd())
			if err != nil {
				log.Printf("Failed to send ping command: %v", err)
				c.PingIntervalTimer.Stop()
				return
			}
			// Make sure we receive a pong in time
			pongTimeoutTimer := time.NewTicker(1 * time.Second)
			go func() {
				for {
					select {
					case tt := <-pongTimeoutTimer.C:
						// If not, try to restart the connection
						log.Printf("Did not receive a pong in time %v", tt)

						pongTimeoutTimer.Stop()
						c.PongTimeoutChannel <- true
						return
					case <-c.PongChannel:
						pongTimeoutTimer.Stop()
						return
					}
				}
			}()
		}
	}
}

// authenticate handles the authentication phase of the Home Assistant Websocket API
func (c *Client) authenticate(msg *IncomingMessage) {
	switch msg.Type {
	case "auth_required":
		log.Print("Sending authntication credentials")
		c.writeChan <- struct {
			Type        string `json:"type"`
			AccessToken string `json:"access_token"`
		}{
			Type:        "auth",
			AccessToken: c.Credentials.AccessToken(),
		}

		return
	case "auth_ok":
		log.Print("authentication succeeded")
		c.Authenticated = true
		c.Ready = true
		close(c.Started)
		return
	case "auth_invalid":
		log.Print("authentication failed")
		c.Authenticated = false
		c.ListenError = NewClientError("Client.authenticate", NotAuthenticatedError)
		c.Close()
	default:
		/* code */
		return
	}
}

// Close closes the websocket client.
func (c *Client) Close() {
	// Compare and Swap to 1 and proceed, return if already 1.
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}

	c.quitWriterChan <- struct{}{}
	close(c.writeChan)
	_ = c.Conn.Close()
}

// TODO: un-export the Conn so that Write methods go through the writer
func (c *Client) writer() {
	for {
		select {
		case msg := <-c.writeChan:
			err := c.Conn.WriteJSON(msg)
			if err != nil {
				log.Printf("failed to write to writeChan: %v", err)
			}

		case <-c.quitWriterChan:
			return
		}
	}
}

// func (c *Client) SubscribeToEventType(eventType string, ch chan *IncomingEventMessage){

// }

// SendCommand sends a command over the websocket connection to Home Assisstant
func (c *Client) SendCommand(command Cmd) error {
	if !c.Authenticated {
		return NotAuthenticatedError
	}
	command.SetID(c.Sequence)

	c.Sequence++
	c.writeChan <- command
	return nil
}
// SendCommandWithCallback Sends a command over the websocket. The callback will be executed when we receive
// a result message with the same Sequence ID. The Callback is executed **once** and will be removed after
// the callback is called.
func (c *Client) SendCommandWithCallback(command Cmd, callback func(message *IncomingResultMessage)) error {
	if !c.Authenticated {
		return NotAuthenticatedError
	}
	c.callbacks[c.Sequence] = callback
	command.SetID(c.Sequence)
	c.Sequence++
	c.writeChan <- command
	return nil
}
