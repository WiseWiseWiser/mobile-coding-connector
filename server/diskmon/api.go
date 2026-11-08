package diskmon

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// cacheTTL collapses concurrent browser polls onto one mount-table read.
const cacheTTL = 2 * time.Second

type sampler struct {
	env Env

	mu       sync.Mutex
	cached   *Snapshot
	cachedAt time.Time
}

var defaultSampler = &sampler{env: defaultEnv()}

// Get returns the cached-or-fresh disk snapshot.
func Get() (*Snapshot, error) {
	return defaultSampler.get()
}

func (s *sampler) get() (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.env.Now()
	if s.cached != nil && now.Sub(s.cachedAt) < cacheTTL {
		return s.cached, nil
	}
	entries, err := s.env.ReadMounts()
	if err != nil {
		return nil, err
	}
	snap := buildSnapshot(entries, now)
	if s.env.LatestIO != nil {
		if io, ok := s.env.LatestIO(); ok {
			snap.IO = &io
		}
	}
	s.cached = snap
	s.cachedAt = now
	return snap, nil
}

// RegisterAPI mounts GET /api/server/disk and the manual space-analysis
// endpoints under /api/server/disk/analysis.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/server/disk", handleDisk)
	registerAnalysisAPI(mux)
	registerCleanupAPI(mux)
}

func handleDisk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap, err := Get()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}
