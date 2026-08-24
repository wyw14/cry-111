package verifycase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/wyw14/cry-111/internal/api"
	"github.com/wyw14/cry-111/internal/flank"
)

func TestOverlapAndShuntRouteReserveAtomically(t *testing.T) {
	app, err := api.NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewServer(app).Handler())
	defer server.Close()
	topology := flank.Topology{
		Revision: 2,
		Main:     map[string][]string{"R14": {"1DG"}, "SH37": {"3DG"}},
		Flank:    map[string][]string{"R14": {"P42"}, "SH37": {"P31"}},
		Overlap:  map[string][]string{"R14": {"5DG"}, "SH37": {"5DG"}},
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
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for _, name := range []string{"R14", "SH37"} {
		wait.Add(1)
		go func(routeName string) {
			defer wait.Done()
			<-start
			body, _ := json.Marshal(map[string]string{"name": routeName})
			response, requestErr := http.Post(server.URL+"/api/routes/request", "application/json", bytes.NewReader(body))
			if requestErr != nil {
				statuses <- 0
				return
			}
			response.Body.Close()
			statuses <- response.StatusCode
		}(name)
	}
	close(start)
	wait.Wait()
	close(statuses)
	created := 0
	for status := range statuses {
		if status == http.StatusCreated {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("expected exactly one complete conflicting route, got %d", created)
	}
}
