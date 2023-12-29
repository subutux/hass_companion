package ws

import (
	"context"
	"time"

	"github.com/subutux/hass_companion/internal/logger"
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
			logger.I().Warn("MonitorConnection Stopped")
			return
		case t := <-PingIntervalTimer.C:
			logger.I().Info("Ping", "t", t)
			err := c.SendCommand(NewPingCmd())
			if err != nil {
				logger.I().Error("Failed to send ping command", "error", err)
				PingIntervalTimer.Stop()
				return
			}
			// Make sure we receive a pong in time
			ctx, cancel := context.WithDeadline(context.Background(),
				time.Now().Add(1*time.Second))
			defer cancel()
			go func() {
				for {
					select {
					case <-ctx.Done():
						if err := ctx.Err(); err != nil {
							logger.I().Error("Did not receive a pong in time %v", "error", err)
							PingIntervalTimer.Stop()
							c.quitPingWatchdog <- struct{}{}
							c.PongTimeoutChannel <- true
						}

						return
					case <-c.PongChannel:
						cancel()
						return
					}
				}
			}()
		}
	}
}
