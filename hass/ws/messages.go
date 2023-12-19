package ws

import (
	"bytes"
	"encoding/json"
	"time"
)

type IncomingMessage struct {
	ID   int64       `json:"id"`
	Type MessageType `json:"type"`
}

func IncomingMessageFromJSON(data []byte) (*IncomingMessage, error) {
	var msg IncomingMessage
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (msg *IncomingMessage) Is(Type MessageType) bool {
	return msg.Type == Type
}
func (msg *IncomingMessage) In(Type ...MessageType) bool {
	for _, val := range Type {
		if val == msg.Type {
			return true
		}
	}
	return false
}

type IncomingResultMessage struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	Success bool   `json:"success"`
	Result  any    `json:"result"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
}

func IncomingResultMessageFromJSON(data []byte) (*IncomingResultMessage, error) {
	var msg IncomingResultMessage
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type IncomingEventMessage struct {
	ID    int64       `json:"id"`
	Type  MessageType `json:"type"`
	Event struct {
		Data      map[string]any `json:"data"`
		EventType string         `json:"event_type"`
	} `json:"event"`
}

func IncomingEventMessageFromJSON(data []byte) (*IncomingEventMessage, error) {
	var msg IncomingEventMessage
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type IncomingPushNotificationMessage struct {
	ID    int64       `json:"id"`
	Type  MessageType `json:"type"`
	Event struct {
		Title   string `json:"title"`
		Message string `json:"message"`
		Target  string `json:"target"`
		Data    struct {
			Actions []struct {
				Action string `json:"action"`
				Title  string `json:"title"`
				Uri    string `json:"uri"`
			} `json:"actions"`
		} `json:"data"`
		HassConfirmId string `json:"hass_confirm_id"`
	} `json:"event"`
}

func IncomingPushNotificationMessageFromJSON(data []byte) (*IncomingPushNotificationMessage, error) {
	var msg IncomingPushNotificationMessage
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type IncomingPongMessage struct {
	ID   int64       `json:"id"`
	Type MessageType `json:"type"`
}

func IncomingPongMessageFromJSON(data []byte) (*IncomingPongMessage, error) {
	var msg IncomingPongMessage
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

type OutgoingCommand struct {
	ID   int64       `json:"id"`
	Type MessageType `json:"type"`
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
