package ws

type Cmd interface {
	SetID(ID int64)
}

type SubscribeToEventsCmd struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	EventType string `json:"event_type,omitempty"`
}

func (c *SubscribeToEventsCmd) SetID(ID int64) {
	c.ID = ID
}

func NewSubscribeToEvents(eventType string) *SubscribeToEventsCmd {
	return &SubscribeToEventsCmd{
		Type:      "subscribe_events",
		EventType: eventType,
	}
}

type Trigger struct {
	Platform string `json:"platform"`
	EntityID string `json:"entity_id"`
	From     string `json:"from"`
	To       string `json:"to"`
}

type SubscribeToTriggerCmd struct {
	ID      int64   `json:"id"`
	Type    string  `json:"type"`
	Trigger Trigger `json:"trigger"`
}

func (c *SubscribeToTriggerCmd) SetID(ID int64) {
	c.ID = ID
}

func NewSubscribeToTriggerCmd(trigger Trigger) SubscribeToTriggerCmd {
	return SubscribeToTriggerCmd{
		Type:    "subscribe_trigger",
		Trigger: trigger,
	}
}

type FireEventCmd struct {
	ID        int64             `json:"id"`
	Type      string            `json:"type"`
	EventType string            `json:"event_type"`
	EventData map[string]string `json:"event_data,omitempty"`
}

func (c *FireEventCmd) SetID(ID int64) {
	c.ID = ID
}

func NewFireEventCmd(EventType string, EventData map[string]string) FireEventCmd {
	return FireEventCmd{
		Type:      "fire_event",
		EventType: EventType,
		EventData: EventData,
	}
}

type CallServiceCmd struct {
	ID          int64             `json:"id"`
	Type        string            `json:"type"`
	Domain      string            `json:"domain"`
	Service     string            `json:"service"`
	ServiceData map[string]string `json:"service_data,omitempty"`
	Target      struct {
		EntityID string `json:"entity_id"`
	} `json:"target,omitempty"`
}

func (c *CallServiceCmd) SetID(ID int64) {
	c.ID = ID
}

func NewCallServiceCmd(Domain, service, target string, ServiceData map[string]string) CallServiceCmd {
	call := CallServiceCmd{
		Type:        "call_service",
		Domain:      Domain,
		Service:     service,
		ServiceData: ServiceData,
	}

	if target != "" {
		call.Target = struct {
			EntityID string "json:\"entity_id\""
		}{
			EntityID: target,
		}
	}

	return call
}

type SubscribeToPushNotificationsChannelCmd struct {
	ID             int64  `json:"id"`
	Type           string `json:"type"`
	WebhookId      string `json:"webhook_id"`
	SupportConfirm bool   `json:"support_confirm"`
}

func (c *SubscribeToPushNotificationsChannelCmd) SetID(ID int64) {
	c.ID = ID
}

func NewSubscribeToPushNotificationsChannelCmd(webhookId string) *SubscribeToPushNotificationsChannelCmd {
	return &SubscribeToPushNotificationsChannelCmd{
		Type:           "mobile_app/push_notification_channel",
		WebhookId:      webhookId,
		SupportConfirm: true,
	}
}

type PingCmd struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

func (c *PingCmd) SetID(ID int64) {
	c.ID = ID
}

func NewPingCmd() *PingCmd {
	return &PingCmd{
		Type: "ping",
	}
}

type GetStatesCmd struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

func (c *GetStatesCmd) SetID(ID int64) {
	c.ID = ID
}

func NewGetStatesCmd() *GetStatesCmd {
	return &GetStatesCmd{
		Type: "get_states",
	}
}

type GetConfigCmd struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

func (c *GetConfigCmd) SetID(ID int64) {
	c.ID = ID
}

func NewGetConfigCmd() *GetConfigCmd {
	return &GetConfigCmd{
		Type: "get_config",
	}
}

type GetWebhookCmd struct {
	ID        int64             `json:"id"`
	Type      string            `json:"type"`
	WebhookID string            `json:"webhook_id"`
	Method    string            `json:"method"`
	Body      string            `json:"body"`
	Headers   map[string]string `json:"headers"`
	Query     string            `json:"query"`
}

func (c *GetWebhookCmd) SetID(ID int64) {
	c.ID = ID
}

func NewGetWebhookCmd(webhookid string, body string) *GetWebhookCmd {
	return &GetWebhookCmd{
		Type: "webhook/handle",
		//WebhookID: webhookid,
		Body:   body,
		Method: "GET",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Query: "",
	}
}


// TODO:
// get_config
// get_services
// get_panels
// [deprecated] camera_thumbnail
// media_player_thumbnail
// validate_config
