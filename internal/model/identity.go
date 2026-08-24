package model

import "github.com/google/uuid"

type Identity struct {
	Value string `json:"value"`
}

func NewIdentity() Identity {
	return Identity{Value: uuid.NewString()}
}

func ParseIdentity(value string) (Identity, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return Identity{}, err
	}
	return Identity{Value: parsed.String()}, nil
}

func (i Identity) String() string {
	return i.Value
}

func (i Identity) Empty() bool {
	return i.Value == ""
}
