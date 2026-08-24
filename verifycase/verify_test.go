package verifycase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wyw14/cry-111/internal/api"
	"github.com/wyw14/cry-111/internal/model"
)

func TestPowerRecoveryWaitsForAllInterlockingDomains(t *testing.T) {
	app, err := api.NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewServer(app).Handler())
	defer server.Close()
	for _, domain := range []string{"point-relay", "signal-lamp", "track-circuit"} {
		response, requestErr := http.Post(server.URL+"/api/power/"+domain+"/state", "application/json", bytes.NewBufferString(`{"state":"offline"}`))
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("power transition for %s returned %d", domain, response.StatusCode)
		}
	}
	routeValue := model.Route{ID: "persisted-r22", Name: "R22", Kind: "train", Phase: model.RouteSignalled, SignalID: "S22", TopologyRevision: 1}
	payload, _ := json.Marshal(map[string]any{"routes": []model.Route{routeValue}})
	response, err := http.Post(server.URL+"/api/recovery/routes", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("recovery published route before all domains were ready: %d", response.StatusCode)
	}
}
