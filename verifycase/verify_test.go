package verifycase

import (
	"testing"
	"time"

	"github.com/wyw14/cry-111/internal/model"
	"github.com/wyw14/cry-111/internal/signal"
)

func TestSignalProofMatchesSelectedAspect(t *testing.T) {
	store := signal.NewStore([]string{"S8"})
	selector := signal.NewSelector(store)
	proof := signal.NewLampProof(store)
	now := time.Unix(500, 0).UTC()
	if _, err := selector.Command("S8", "route-8", model.AspectProceed, now); err != nil {
		t.Fatal(err)
	}
	if _, err := selector.Observe("S8", model.AspectStop, model.AspectStop, 120, now); err != nil {
		t.Fatal(err)
	}
	state, err := proof.Evaluate("S8", now)
	if err != nil {
		t.Fatal(err)
	}
	if state.Proved || state.AspectProved() {
		t.Fatalf("wrong selected lamp satisfied proceed proof: %+v", state)
	}
}
