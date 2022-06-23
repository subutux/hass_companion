package hass

import "time"

type EventMessage struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Event struct {
		Data      interface{} `json:"data"`
		EventType string      `json:"event_type"`
		TimeFired time.Time   `json:"time_fired"`
		Origin    string      `json:"origin"`
		Context   struct {
			ID       string      `json:"id"`
			ParentID interface{} `json:"parent_id"`
			UserID   string      `json:"user_id"`
		} `json:"context"`
	} `json:"event"`
}
