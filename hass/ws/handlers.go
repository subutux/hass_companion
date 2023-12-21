package ws

import "log"

func (c *Client) handlePong(message []byte) {
	result, jsonErr := IncomingPongMessageFromJSON(message)
	if jsonErr != nil {
		log.Printf("Failed to decode result from json: %s", jsonErr)
	} else {
		c.PongChannel <- result
	}
}

func (c *Client) handleAuth(message []byte) error {
	result, err := IncomingMessageFromJSON(message)
	if err != nil {
		log.Printf("Failed to decode result from json: %s", err)
		return err
	} else {
		c.authenticate(result)
	}
	return nil
}

func (c *Client) handleEvent(message []byte) error {
	// First, try to decode it as an push notification
	notification, jsonErr := IncomingPushNotificationMessageFromJSON(message)
	if jsonErr != nil {
		log.Printf("Failed to decode event from json: %s", jsonErr)
		return jsonErr
	} else if notification.Event.Message != "" {
		log.Printf("received push notification %v", string(message))

		c.PushNotificationChannel <- notification

	}
	// then continue to event processing
	event, jsonErr := IncomingEventMessageFromJSON(message)
	if jsonErr != nil {
		log.Printf("Failed to decode event from json: %s", jsonErr)
		return jsonErr
	} else {
		log.Printf("received %s event", event.Event.EventType)
		c.EventChannel <- event

	}
	return nil
}

func (c *Client) handleResult(message []byte) error {
	msg, err := IncomingResultMessageFromJSON(message)
	if err != nil {
		log.Printf("Failed to decode result from json: %s", err)
		return err
	} else {
		log.Printf("received result with error %v and success %v", msg.Error, msg.Success)
		// first check if we have a callback set for this id
		cb, ok := c.callbacks[msg.ID]
		if ok {
			log.Printf("Calling callback for id %v", msg.ID)
			cb(msg)
			// Delete the callback afterwards
			delete(c.callbacks, msg.ID)
		} else {
			c.ResultChannel <- msg
		}
	}
	return nil
}
