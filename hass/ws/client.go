package ws

import (
	"bytes"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
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

		switch msg.Type {
		case MessageTypeEvent:
			c.handleEvent(raw_message)
		case MessageTypeResult:
			c.handleResult(raw_message)
		case MessageTypeAuthRequired, MessageTypeAuthOK, MessageTypeAuthInvalid:
			c.handleAuth(raw_message)
		case MessageTypePong:
			c.handlePong(raw_message)
		default:
			log.Printf("Unknown message type: %v for message %v", msg.Type, string(raw_message))
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
	c.writeChan <- command
	return nil
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
