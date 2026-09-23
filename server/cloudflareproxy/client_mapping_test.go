package cloudflareproxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeEdge is a minimal stand-in for the edge's mapping API. It reports a
// conflict while an existing mapping is registered, exactly like the real one.
type fakeEdge struct {
	mu          sync.Mutex
	hostname    string
	existingWS  int
	present     bool
	posts       int
	deletes     []string
	sawConflict bool
}

func (f *fakeEdge) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		switch r.Method {
		case http.MethodPost:
			f.posts++
			if f.present {
				f.sawConflict = true
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "hostname already mapped"})
				return
			}
			f.present = true
			_ = json.NewEncoder(w).Encode(MappingView{
				ID:       "map-new",
				Hostname: f.hostname,
				WS:       0,
				URL:      "https://" + f.hostname,
				DialURL:  "wss://edge.example/dial?m=map-new&t=secret",
			})
		case http.MethodGet:
			mappings := []MappingView{}
			if f.present {
				mappings = append(mappings, MappingView{
					ID:       "map-old",
					Hostname: f.hostname,
					WS:       f.existingWS,
					URL:      "https://" + f.hostname,
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"mappings": mappings})
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			f.deletes = append(f.deletes, id)
			f.present = false
			_ = json.NewEncoder(w).Encode(map[string]string{"deleted": id})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func newFakeEdge(t *testing.T, existingWS int) (*fakeEdge, *APIClient) {
	t.Helper()
	edge := &fakeEdge{hostname: "app.example.com", existingWS: existingWS, present: true}
	ts := httptest.NewServer(edge.handler())
	t.Cleanup(ts.Close)
	return edge, &APIClient{BaseURL: ts.URL, Token: "secret"}
}

func TestAddMappingReclaimsDeadMapping(t *testing.T) {
	edge, cli := newFakeEdge(t, 0)

	view, err := cli.AddMapping("app.example.com")
	if err != nil {
		t.Fatalf("AddMapping() error = %v, want the stale mapping reclaimed", err)
	}
	if view.ID != "map-new" {
		t.Fatalf("AddMapping() view = %#v, want the freshly registered mapping", view)
	}
	if len(edge.deletes) != 1 || edge.deletes[0] != "map-old" {
		t.Fatalf("deletes = %v, want the orphaned map-old removed", edge.deletes)
	}
	if edge.posts != 2 {
		t.Fatalf("posts = %d, want the add retried once after reclaiming", edge.posts)
	}
}

func TestAddMappingKeepsConflictForLiveMapping(t *testing.T) {
	edge, cli := newFakeEdge(t, 4)

	_, err := cli.AddMapping("app.example.com")
	if err == nil {
		t.Fatal("AddMapping() must fail while another origin holds live dial sockets")
	}
	if !strings.Contains(err.Error(), "already mapped") {
		t.Fatalf("error = %v, want it to mention the conflict", err)
	}
	if !strings.Contains(err.Error(), "live origin") {
		t.Fatalf("error = %v, want it to explain that a live origin owns the hostname", err)
	}
	if len(edge.deletes) != 0 {
		t.Fatalf("deletes = %v, want a live mapping left untouched", edge.deletes)
	}
	if edge.posts != 1 {
		t.Fatalf("posts = %d, want no retry when the mapping is live", edge.posts)
	}
}

func TestAddMappingDoesNotReclaimOnOtherErrors(t *testing.T) {
	var deletes int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "boom"})
		case r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"mappings": []MappingView{}})
		default:
			deletes++
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	if _, err := cli.AddMapping("app.example.com"); err == nil {
		t.Fatal("AddMapping() must surface a server error")
	}
	if deletes != 0 {
		t.Fatalf("deletes = %d, want no reclaim attempt for a non-conflict error", deletes)
	}
}

func TestAddMappingReclaimsOnlyTheRequestedHostname(t *testing.T) {
	var deleted []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "hostname already mapped"})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"mappings": []MappingView{
				{ID: "map-other", Hostname: "other.example.com", WS: 0},
			}})
		case http.MethodDelete:
			deleted = append(deleted, r.URL.Query().Get("id"))
			_ = json.NewEncoder(w).Encode(map[string]string{"deleted": "map-other"})
		}
	}))
	defer ts.Close()

	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	if _, err := cli.AddMapping("app.example.com"); err == nil {
		t.Fatal("AddMapping() must fail when no mapping matches the hostname")
	}
	if len(deleted) != 0 {
		t.Fatalf("deleted = %v, want other hostnames left alone", deleted)
	}
}
