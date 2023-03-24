package ws

import (
	"bytes"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/hass/auth"
)

const avgReadMsgSizeBytes = 1024

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

	ListenError ClientError
}

func NewClient(url string, credentials *auth.Credentials) (*Client, error) {
	dailer := websocket.DefaultDialer
	conn, _, err := dailer.Dial(url, nil)
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
	go func() {
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

			pp.Println(msg)

			if msg.Is("event") {

				// First, try to decode it as an push notification
				notification, jsonErr := IncomingPushNotificationMessageFromJSON(bytes.NewReader(buf.Bytes()))
				if jsonErr != nil {
					log.Printf("Failed to decode event from json: %s", jsonErr)
				} else if notification.Event.Message != "" {
					log.Printf("received push notification")
					c.PushNotificationChannel <- notification
				}
				// then continue to event processing
				event, jsonErr := IncomingEventMessageFromJSON(bytes.NewReader(buf.Bytes()))
				if jsonErr != nil {
					log.Printf("Failed to decode event from json: %s", jsonErr)
				} else {
					log.Printf("received event for entity %v", event.Event.Data.EntityID)
					c.EventChannel <- event
				}
				continue
			}

			if msg.Is("result") {
				result, jsonErr := IncomingResultMessageFromJSON(bytes.NewReader(buf.Bytes()))
				if jsonErr != nil {
					log.Printf("Failed to decode result from json: %s", jsonErr)
				} else {
					pp.Println(result)
					log.Printf("received result with error %v and success %v", result.Error, result.Success)
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
	}()
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
			c.SendCommand(NewPingCmd())
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
		c.ListenError = NewClientError("Client.authenticate", errors.New("Authentication invalid"))
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
	c.Conn.Close()
}

// TODO: un-export the Conn so that Write methods go through the writer
func (c *Client) writer() {
	for {
		select {
		case msg := <-c.writeChan:
			c.Conn.WriteJSON(msg)

		case <-c.quitWriterChan:
			return
		}
	}
}

// SendCommand sends a command over the websocket connection to Home Assisstant
func (c *Client) SendCommand(command Cmd) error {
	if !c.Authenticated {
		return errors.New("Not authenticated")
	}
	command.SetID(c.Sequence)

	c.Sequence++
	pp.Println(command)
	c.writeChan <- command
	return nil
}
