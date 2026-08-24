package verifycase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wyw14/cry-111/internal/api"
	"github.com/wyw14/cry-111/internal/model"
)

func TestAdjacentAxleResetsCommitSharedBoundaryAtomically(t *testing.T) {
	app, err := api.NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(400, 0).UTC()
	for id, count := range map[string]int64{"B11": 5, "B12": 3, "B13": 7} {
		if _, err := app.Axles.Count(id, count, now); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(api.NewServer(app).Handler())
	defer server.Close()
	body, _ := json.Marshal(map[string]string{
		"first":      "12AC",
		"second":     "13AC",
		"operator_a": "operator-a",
		"operator_b": "operator-b",
	})
	response, err := http.Post(server.URL+"/api/axle/resets/adjacent", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("adjacent reset returned %d", response.StatusCode)
	}
	var payload struct {
		Sections []model.TrackState `json:"sections"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Sections) != 2 {
		t.Fatalf("expected two published track sections, got %d", len(payload.Sections))
	}
	for _, section := range payload.Sections {
		if section.Occupied {
			t.Fatalf("adjacent reset published contradictory occupied section: %+v", section)
		}
	}
}
