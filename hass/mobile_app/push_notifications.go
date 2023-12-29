package mobile_app

import (
	"log"

	"github.com/godbus/dbus/v5"
	"github.com/subutux/hass_companion/hass/ws"
)

// EnableWebsocketPushNotifications sends a command to Home Assistant that
// This client supports Push notifications over websockets.
func (m *MobileApp) EnableWebsocketPushNotifications() {
	m.ws.SendCommand(ws.NewSubscribeToPushNotificationsChannelCmd(m.Registration.WebhookID))
}

// WatchForPushNotifications calls the callback function with the notification
// whenever there is a push notification received on the websocket connection.
func (m *MobileApp) WatchForPushNotifications(callback func(notification *ws.IncomingPushNotificationMessage)) {
	for notification := range m.ws.PushNotificationChannel {
		log.Printf("NOTIFY %v", notification)
		callback(notification)
	}
}

func (m *MobileApp) FreedesktopNotifier(notification *ws.IncomingPushNotificationMessage) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return
	}

	obj := conn.Object("org.freedesktop.Notifications", dbus.ObjectPath("/org/freedesktop/Notifications"))

	actions := []string{}
	for _, a := range notification.Event.Data.Actions {
		actions = append(actions, a.Action, a.Title)
	}

	log.Printf("--> Actions %v", actions)

	// Timeouts in Freedesktop Notifications are in ms
	timeout := notification.Event.Data.Timeout * 1000
	if timeout == 0 {
		// Default to system timeout if timeout is zero.
		timeout = -1
	}

	call := obj.Call("org.freedesktop.Notifications.Notify", 0, "HASS Companion",
		uint32(0), "",
		notification.Event.Title, notification.Event.Message,
		actions, map[string]dbus.Variant{}, timeout)

	if call.Err != nil {
		log.Printf("Error sending notification: %v", call.Err)
		return
	}
	// confirm
	if notification.Event.HassConfirmId != "" {
		m.ws.SendCommand(
			ws.NewOutgoingPushNotificationConfirmation(
				m.Registration.WebhookID,
				notification.Event.HassConfirmId),
		)
	}

	//notifID := call.Body[0].(int32)

	// Wait for actions
	if len(actions) > 0 {
		go func() {
			conn.AddMatchSignal(
				dbus.WithMatchObjectPath("/org/freedesktop/Notifications"),
				dbus.WithMatchInterface("org.freedesktop.Notifications"),
			)
			c := make(chan *dbus.Signal, 10)
			conn.Signal(c)
			log.Print("Waiting for action")
			for v := range c {
				log.Printf("---> ACTION:: %v", v)
				if len(v.Body) == 2 && v.Name == "org.freedesktop.Notifications.ActionInvoked" {

					// TODO handle action
					// - if uri is set, handle uri

					eventCmd := ws.NewFireEventCmd("mobile_app_notification_action", map[string]string{
						"action": v.Body[1].(string),
						// For a proper implementation, see https://github.com/chipaca/snappy/blob/1f06296fe8adebee7eaef50e21875fd37ac19f04/desktop/notification/fdo.go#L186
						// https://specifications.freedesktop.org/notification-spec/notification-spec-latest.html table 8
					})
					m.ws.SendCommand(&eventCmd)
				}
				conn.RemoveSignal(c)

			}

		}()
	}

}

/**
 * method call time=1702973466.628632 sender=:1.506 -> destination=:1.23 serial=8 path=/org/freedesktop/Notifications; interface=org.freedesktop.Notifications; member=GetCapabilities
method return time=1702973466.629114 sender=:1.23 -> destination=:1.506 serial=3982 reply_serial=8
   array [
      string "body"
      string "body-hyperlinks"
      string "body-markup"
      string "body-images"
      string "icon-static"
      string "actions"
      string "persistence"
      string "inline-reply"
      string "x-kde-urls"
      string "x-kde-origin-name"
      string "x-kde-display-appname"
      string "inhibitions"
   ]
method call time=1702973466.629169 sender=:1.506 -> destination=:1.23 serial=9 path=/org/freedesktop/Notifications; interface=org.freedesktop.Notifications; member=Notify
   string "notify-send"
   uint32 0
   string "/home/svancampenhout/Pictures/2020-11-27_21-50.png"
   string "notif test"
   string "this is the body <img src="file://home/svancampenhout/Pictures/2020-11-27_21-50.png">"
   array [
      string "Test"
      string "⚒"
   ]
   array [
      dict entry(
         string "urgency"
         variant             byte 1
      )
      dict entry(
         string "sender-pid"
         variant             int64 85207
      )
   ]
   int32 -1
   **/
