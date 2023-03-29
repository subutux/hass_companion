package ws

import (
	"encoding/json"
	"io"
	"time"
)

type IncomingMessage struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

func IncomingMessageFromJSON(data io.Reader) (*IncomingMessage, error) {
	var msg IncomingMessage
	if err := json.NewDecoder(data).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (msg *IncomingMessage) Is(Type string) bool {
	return msg.Type == Type
}
func (msg *IncomingMessage) In(Type ...string) bool {
	for _, val := range Type {
		if val == msg.Type {
			return true
		}
	}
	return false
}

type IncomingResultMessage struct {
	ID      int64          `json:"id"`
	Type    string         `json:"type"`
	Success bool           `json:"success"`
	Result  any `json:"result"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
}

func IncomingResultMessageFromJSON(data io.Reader) (*IncomingResultMessage, error) {
	var msg IncomingResultMessage
	if err := json.NewDecoder(data).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type IncomingEventMessage struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Event struct {
		Data      map[string]any `json:"data"`
		EventType string         `json:"event_type"`
	} `json:"event"`
}

func IncomingEventMessageFromJSON(data io.Reader) (*IncomingEventMessage, error) {
	var msg IncomingEventMessage
	if err := json.NewDecoder(data).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type IncomingPushNotificationMessage struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Event struct {
		Title       string `json:"title"`
		Message       string `json:"message"`
		HassConfirmId string `json:"hass_confirm_id"`
	} `json:"event"`
}

func IncomingPushNotificationMessageFromJSON(data io.Reader) (*IncomingPushNotificationMessage, error) {
	var msg IncomingPushNotificationMessage
	if err := json.NewDecoder(data).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type IncomingPongMessage struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

func IncomingPongMessageFromJSON(data io.Reader) (*IncomingPongMessage, error) {
	var msg IncomingPongMessage
	if err := json.NewDecoder(data).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type OutgoingCommand struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type State struct {
	EntityID    string         `json:"entity_id"`
	LastChanged time.Time      `json:"last_changed"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
	LastUpdated time.Time      `json:"last_updated"`
	Context     struct {
		ID       string      `json:"id"`
		ParentID interface{} `json:"parent_id"`
		UserID   string      `json:"user_id"`
	} `json:"context"`
}
