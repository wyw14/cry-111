package verifycase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/wyw14/cry-111/internal/api"
	"github.com/wyw14/cry-111/internal/flank"
	"github.com/wyw14/cry-111/internal/model"
)

func TestTopologyChangeRebuildsRouteFlankProtection(t *testing.T) {
	app, err := api.NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app.Resolver.Resolve(model.Route{Name: "R14"})
	server := httptest.NewServer(api.NewServer(app).Handler())
	defer server.Close()
	topology := flank.Topology{
		Revision: 2,
		Main:     map[string][]string{"R14": {"1DG", "3DG", "5DG"}},
		Flank:    map[string][]string{"R14": {"P31", "P37", "P42"}},
		Overlap:  map[string][]string{"R14": {"OV14"}},
	}
	payload, _ := json.Marshal(topology)
	request, _ := http.NewRequest(http.MethodPut, server.URL+"/api/topology", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("topology update returned %d", response.StatusCode)
	}
	response, err = http.Post(server.URL+"/api/routes/request", "application/json", bytes.NewBufferString(`{"name":"R14"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("route request returned %d", response.StatusCode)
	}
	var routeValue model.Route
	if err := json.NewDecoder(response.Body).Decode(&routeValue); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(routeValue.FlankPoints, "P31") {
		t.Fatalf("new flank point missing from route plan: %v", routeValue.FlankPoints)
	}
	pointState, ok := app.Points.Get("P31")
	if !ok || pointState.LockedBy != routeValue.ID {
		t.Fatalf("new flank point was not locked by route: %+v", pointState)
	}
}
