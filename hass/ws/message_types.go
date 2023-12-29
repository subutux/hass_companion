package ws

type MessageType string

const (
	MessageTypeAuthRequired      MessageType = "auth_required"
	MessageTypeAuth                          = "auth"
	MessageTypeAuthOK                        = "auth_ok"
	MessageTypeAuthInvalid                   = "auth_invalid"
	MessageTypeSubscribeEvents               = "subscribe_events"
	MessageTypeSubscribeTrigger              = "subscribe_trigger"
	MessageTypeEvent                         = "event"
	MessageTypePing                          = "ping"
	MessageTypePong                          = "pong"
	MessageTypeGetStates                     = "get_states"
	MessageTypeGetServices                   = "get_services"
	MessageTypeGetConfig                     = "get_config"
	MessageTypeGetEntityRegistry             = "config/entity_registry/list"
	MessageTypeGetDeviceRegistry             = "config/device_registry/list"
	MessageTypeResult                        = "result"
)

func (mt MessageType) Valid() bool {
	switch mt {
	case MessageTypeAuthRequired,
		MessageTypeAuth,
		MessageTypeAuthOK,
		MessageTypeAuthInvalid,
		MessageTypeSubscribeEvents,
		MessageTypeSubscribeTrigger,
		MessageTypeEvent,
		MessageTypePing,
		MessageTypePong,
		MessageTypeGetStates,
		MessageTypeGetServices,
		MessageTypeGetEntityRegistry,
		MessageTypeGetDeviceRegistry,
		MessageTypeResult:
		return true
	}

	return false
}
