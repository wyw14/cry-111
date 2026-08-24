package verifycase

import (
	"testing"
	"time"

	"github.com/wyw14/cry-111/internal/approach"
)

type adjustableClock struct {
	wall    time.Time
	elapsed time.Duration
}

func (c *adjustableClock) WallNow() time.Time {
	return c.wall
}

func (c *adjustableClock) MonotonicElapsed() time.Duration {
	return c.elapsed
}

func TestApproachLockIgnoresWallClockCorrection(t *testing.T) {
	clock := &adjustableClock{wall: time.Unix(300, 0).UTC()}
	timer := approach.NewTimer(clock)
	if _, err := timer.Start("route-5", 30*time.Second); err != nil {
		t.Fatal(err)
	}
	clock.wall = clock.wall.Add(12 * time.Second)
	remaining, err := timer.Remaining("route-5")
	if err != nil {
		t.Fatal(err)
	}
	if remaining != 30*time.Second {
		t.Fatalf("wall clock correction changed safety duration: %s", remaining)
	}
	if timer.Expired("route-5") {
		t.Fatal("approach lock expired after wall clock correction")
	}
}
