package ws

import (
	"github.com/subutux/hass_companion/internal/logger"
)

func (c *Client) handlePong(message []byte) {
	log := logger.I()
	result, jsonErr := IncomingPongMessageFromJSON(message)
	if jsonErr != nil {
		log.Error("Failed to decode result from json", "error", jsonErr)
	} else {
		c.PongChannel <- result
	}
}

func (c *Client) handleAuth(message []byte) error {
	log := logger.I()
	result, err := IncomingMessageFromJSON(message)
	if err != nil {
		log.Error("Failed to decode result from json", "error", err)
		return err
	} else {
		c.authenticate(result)
	}
	return nil
}

func (c *Client) handleEvent(message []byte) error {
	log := logger.I()
	// First, try to decode it as an push notification
	notification, jsonErr := IncomingPushNotificationMessageFromJSON(message)
	if jsonErr != nil {
		log.Error("Failed to decode result from json", "error", jsonErr)
		return jsonErr
	} else if notification.Event.Message != "" {
		log.Info("received push notification", "message", string(message))
		c.PushNotificationChannel <- notification
	}
	// then continue to event processing
	event, jsonErr := IncomingEventMessageFromJSON(message)
	if jsonErr != nil {
		log.Error("Failed to decode result from json", "error", jsonErr)
		return jsonErr
	} else {
		log.Info("received event", "type", event.Event.EventType)
		c.EventChannel <- event
	}
	return nil
}

func (c *Client) handleResult(message []byte) error {
	log := logger.I()
	msg, err := IncomingResultMessageFromJSON(message)
	if err != nil {
		log.Error("Failed to decode result from json", "error", err)
		return err
	} else {
		log.Info("received result", "error", msg.Error, "success", msg.Success)
		// first check if we have a callback set for this id
		cb, ok := c.callbacks[msg.ID]
		if ok {
			log.Debug("Calling callback", "callback id", msg.ID)
			cb(msg)
			// Delete the callback afterwards
			delete(c.callbacks, msg.ID)
		} else {
			c.ResultChannel <- msg
		}
	}
	return nil
}
