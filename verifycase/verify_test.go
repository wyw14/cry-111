package verifycase

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-111/internal/api"
)

func TestEmergencyCancelWaitsForSignalDarkProof(t *testing.T) {
	app, err := api.NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	routeValue, err := app.Routes.Request("R14", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, requestErr := app.Emergency.Request(ctx, routeValue.ID, time.Now().UTC())
		result <- requestErr
	}()
	time.Sleep(40 * time.Millisecond)
	if app.Resources.Count() == 0 {
		t.Fatal("route resources released before signal dark proof")
	}
	if err := app.Emergency.ConfirmDark(routeValue.SignalID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}
