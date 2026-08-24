package model

import "time"

type Event struct {
	ID       string         `json:"id"`
	RoundID  string         `json:"round_id"`
	Kind     string         `json:"kind"`
	EntityID string         `json:"entity_id"`
	At       time.Time      `json:"at"`
	Sequence uint64         `json:"sequence"`
	Payload  map[string]any `json:"payload"`
}

func NewEvent(kind, entityID string, payload map[string]any) Event {
	return Event{
		ID:       NewIdentity().String(),
		RoundID:  NewIdentity().String(),
		Kind:     kind,
		EntityID: entityID,
		At:       time.Now().UTC(),
		Payload:  clonePayload(payload),
	}
}

func (e Event) Clone() Event {
	e.Payload = clonePayload(e.Payload)
	return e
}

func clonePayload(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
