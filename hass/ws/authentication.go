package ws

import "log"

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
