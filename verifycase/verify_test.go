package verifycase

import (
	"testing"
	"time"

	"github.com/wyw14/cry-111/internal/crossing"
)

func TestCrossingRecloseRequiresFreshGateDownProof(t *testing.T) {
	gate := crossing.NewGate("LC1")
	controller := crossing.NewController(gate)
	now := time.Unix(600, 0).UTC()
	first, err := controller.Close("route-first", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.ProveDown(first.ID, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.Open(first.ID, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	second, err := controller.Reclose(first.ID, "route-second", now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if controller.Protected(second.ID) {
		t.Fatal("new crossing session inherited a previous gate-down proof")
	}
}
