package ws

import (
	"net/url"
)

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
	return wsUrl, nil

}
