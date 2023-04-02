package mobile_app

import "github.com/subutux/hass_companion/hass/ws"

// EnableWebsocketPushNotifications sends a command to Home Assistant that
// This client supports Push notifications over websockets.
func (m *MobileApp) EnableWebsocketPushNotifications() {
	m.ws.SendCommand(ws.NewSubscribeToPushNotificationsChannelCmd(m.Registration.WebhookID))
}

// WatchForPushNotifications calls the callback function with the notification
// whenever there is a push notification received on the websocket connection.
func (m *MobileApp) WatchForPushNotifications(callback func(notification *ws.IncomingPushNotificationMessage)) {
	for notification := range m.ws.PushNotificationChannel {
		callback(notification)
	}
}
