package ws

import (
	"bytes"
	"errors"
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

	PongChannel        chan *IncomingPongMessage
	PongTimeoutChannel chan bool
	quitPingWatchdog   chan struct{}

	quitWriterChan chan struct{}
	resetTimerChan chan struct{}
	closed         int32

	writerRunning bool

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
		raw_message := buf.Bytes()
		msg, jsonErr := IncomingMessageFromJSON(raw_message)
		if jsonErr != nil {
			log.Printf("Failed to decode from json: %s", jsonErr)
			continue
		}

		if msg.Is(MessageTypeEvent) {

			// First, try to decode it as an push notification
			notification, jsonErr := IncomingPushNotificationMessageFromJSON(raw_message)
			if jsonErr != nil {
				log.Printf("Failed to decode event from json: %s", jsonErr)
			} else if notification.Event.Message != "" {
				log.Printf("received push notification %v", buf.String())

				c.PushNotificationChannel <- notification
				continue
			}
			// then continue to event processing
			event, jsonErr := IncomingEventMessageFromJSON(raw_message)
			if jsonErr != nil {
				log.Printf("Failed to decode event from json: %s", jsonErr)
			} else {
				log.Printf("received %s event", event.Event.EventType)
				c.EventChannel <- event
				continue
			}
		}

		if msg.Is(MessageTypeResult) {
			result, jsonErr := IncomingResultMessageFromJSON(raw_message)
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

		if msg.In(MessageTypeAuthRequired, MessageTypeAuthOK, MessageTypeAuthInvalid) {
			result, jsonErr := IncomingMessageFromJSON(raw_message)
			if jsonErr != nil {
				log.Printf("Failed to decode result from json: %s", jsonErr)
			} else {
				c.authenticate(result)
			}
			continue
		}

		if msg.Is(MessageTypePong) {
			result, jsonErr := IncomingPongMessageFromJSON(raw_message)
			if jsonErr != nil {
				log.Printf("Failed to decode result from json: %s", jsonErr)
			} else {
				c.PongChannel <- result
			}
			continue
		}
	}
}

// Redail tries to reconnect the websocket without closing all channels
// keeping the application running. This is needed for when the connection
// to Home Assistant is lost and we want to try to reconnect.
func (c *Client) Redial() error {
	log.Println("Redialing")
	if c.writerRunning {

		c.quitWriterChan <- struct{}{}
	}
	c.Conn.Close()
	server, err := detectWebsocketUrl(c.Credentials.Server)
	if err != nil {
		return err
	}
	dailer := websocket.DefaultDialer
	dailer.HandshakeTimeout = 5 * time.Second
	conn, _, err := dailer.Dial(server.String(), nil)
	if err != nil {
		return err
	}

	log.Println("Setting up new connection")
	c.Conn = conn
	c.Started = make(chan struct{})
	go c.writer()

	return nil
}

// MonitorConnection periodically sends pings over the websocket connection
// to home assistant and expects a pong message back within one second.
// If we did not received a pong in time, a bool will be posted to the Client.PongTimeoutChannel
// to indicate that there is a problem with the connection.
func (c *Client) MonitorConnection() {
	PingIntervalTimer := time.NewTicker(5 * time.Second)
	// Periodically send a Ping
	for {
		select {
		case <-c.quitPingWatchdog:
			log.Println("MonitorConnection Stopped")
			return
		case t := <-PingIntervalTimer.C:
			log.Printf("Ping at %v", t)
			err := c.SendCommand(NewPingCmd())
			if err != nil {
				log.Printf("Failed to send ping command: %v", err)
				PingIntervalTimer.Stop()
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
						PingIntervalTimer.Stop()
						pongTimeoutTimer.Stop()
						c.quitPingWatchdog <- struct{}{}
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
	case MessageTypeAuthRequired:
		log.Print("Sending authentication credentials")
		c.writeChan <- struct {
			Type        string `json:"type"`
			AccessToken string `json:"access_token"`
		}{
			Type:        "auth",
			AccessToken: c.Credentials.AccessToken(),
		}

		return
	case MessageTypeAuthOK:
		log.Print("authentication succeeded")
		c.Authenticated = true
		c.Ready = true
		close(c.Started)
		return
	case MessageTypeAuthInvalid:
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
	// Compare and Swap to 1 and proceed, return if already 1. (means that we
	// are already closed (or in process of))
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}
	close(c.EventChannel)
	close(c.ResultChannel)
	close(c.PushNotificationChannel)
	close(c.quitPingWatchdog)
	close(c.PongTimeoutChannel)
	close(c.resetTimerChan)
	if !c.Ready {
		close(c.Started)
	}

	c.quitWriterChan <- struct{}{}
	close(c.writeChan)
	_ = c.Conn.Close()
}

func (c *Client) writer() {

	c.writerRunning = true
	for {
		select {
		case msg := <-c.writeChan:
			err := c.Conn.WriteJSON(msg)
			if err != nil {
				log.Printf("failed to write to writeChan: %v", err)
			}

		case <-c.quitWriterChan:
			c.writerRunning = false
			return
		}
	}
}

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
	pp.Print(command)
	c.writeChan <- command
	return nil
}
