package model

import "time"

type PointPosition string

const (
	PointNormal  PointPosition = "normal"
	PointReverse PointPosition = "reverse"
	PointUnknown PointPosition = "unknown"
)

type PointState struct {
	ID             string        `json:"id"`
	Commanded      PointPosition `json:"commanded"`
	Detected       PointPosition `json:"detected"`
	Closed         bool          `json:"closed"`
	MotorRunning   bool          `json:"motor_running"`
	LockedBy       string        `json:"locked_by,omitempty"`
	ProofValid     bool          `json:"proof_valid"`
	ProofRevision  uint64        `json:"proof_revision"`
	DetectionSince time.Time     `json:"detection_since"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type TrackState struct {
	ID             string    `json:"id"`
	Occupied       bool      `json:"occupied"`
	ReservedBy     string    `json:"reserved_by,omitempty"`
	Direction      string    `json:"direction,omitempty"`
	LastTransition time.Time `json:"last_transition"`
	StableSince    time.Time `json:"stable_since"`
}

type SignalAspect string

const (
	AspectDark    SignalAspect = "dark"
	AspectStop    SignalAspect = "stop"
	AspectCall    SignalAspect = "call-on"
	AspectProceed SignalAspect = "proceed"
)

type SignalState struct {
	ID               string       `json:"id"`
	Commanded        SignalAspect `json:"commanded"`
	Selected         SignalAspect `json:"selected"`
	Displayed        SignalAspect `json:"displayed"`
	CurrentMilliAmps int          `json:"current_milliamps"`
	Proved           bool         `json:"proved"`
	DarkProved       bool         `json:"dark_proved"`
	RouteID          string       `json:"route_id,omitempty"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

func (p PointState) PositionProved() bool {
	return p.Closed && !p.MotorRunning && p.Commanded == p.Detected && p.ProofValid
}

func (s SignalState) AspectProved() bool {
	return s.Proved && s.Commanded == s.Selected && s.Commanded == s.Displayed
}
