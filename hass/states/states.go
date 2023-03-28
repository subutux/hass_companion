package states

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/k0kubun/pp/v3"
	"github.com/subutux/hass_companion/hass/ws"
)

type Store struct {
	mu     sync.Mutex
	States []ws.State
}

func NewStore(states []ws.State) *Store {
	return &Store{
		mu:     sync.Mutex{},
		States: states,
	}
}

type ChangeEvent struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Event struct {
		Data struct {
			EntityID string    `json:"entity_id"`
			NewState *ws.State `json:"new_state"`
			OldState *ws.State `json:"old_state"`
		} `json:"data"`
		EventType string `json:"event_type"`
	} `json:"event"`
}

func NewChangeEventFromIncomingEventMessage(msg *ws.IncomingEventMessage) (*ChangeEvent, error) {
	var changeEvent ChangeEvent
	raw, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	pp.Println(string(raw))

	err = json.Unmarshal(raw, &changeEvent)
	if err != nil {
		return nil, err
	}

	return &changeEvent, err
}

func (m *Store) FindByEntityId(id string) (*ws.State, int) {
	for idx, state := range m.States {
		if state.EntityID == id {
			return &state, idx
		}
	}
	return nil, -1
}

func (s *Store) HandleStateChanged(changeEvent *ChangeEvent) error {
	// New state
	if changeEvent.Event.Data.OldState == nil {
		s.mu.Lock()
		s.States = append(s.States, *changeEvent.Event.Data.NewState)
		s.mu.Unlock()
		return nil
	}

	entity, idx := s.FindByEntityId(changeEvent.Event.Data.OldState.EntityID)
	if entity == nil {
		return fmt.Errorf("Cannot find entity %s in store", changeEvent.Event.Data.OldState.EntityID)
	}
	// Is removed
	if changeEvent.Event.Data.NewState == nil {
		s.mu.Lock()
		s.States = append(s.States[:idx], s.States[idx+1:]...)
		s.mu.Unlock()
		return nil
	}

	// Update state
	s.mu.Lock()
	s.States[idx] = *changeEvent.Event.Data.NewState
	s.mu.Unlock()

	return nil
}
