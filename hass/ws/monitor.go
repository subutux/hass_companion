package ws

import (
	"log"
	"time"
)

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
