package model

import "time"

type Incident struct {
	ID             string     `json:"id"`
	Severity       string     `json:"severity"`
	Component      string     `json:"component"`
	EntityID       string     `json:"entity_id"`
	Message        string     `json:"message"`
	Active         bool       `json:"active"`
	RaisedAt       time.Time  `json:"raised_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

func NewIncident(severity, component, entityID, message string) Incident {
	return Incident{
		ID:        NewIdentity().String(),
		Severity:  severity,
		Component: component,
		EntityID:  entityID,
		Message:   message,
		Active:    true,
		RaisedAt:  time.Now().UTC(),
	}
}

func (i Incident) Acknowledge(at time.Time) Incident {
	i.Active = false
	i.AcknowledgedAt = &at
	return i
}
