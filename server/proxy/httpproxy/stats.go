package httpproxy

import (
	"sync"
	"time"
)

// ConnInfo is one proxied connection as the detail page renders it.
type ConnInfo struct {
	ID     uint64 `json:"id"`
	Client string `json:"client"`
	Target string `json:"target"`
	Method string `json:"method"` // GET / POST / CONNECT
	Up     int64  `json:"up"`
	Down   int64  `json:"down"`
	Since  string `json:"since"` // RFC3339
	Secs   int64  `json:"secs"`
	State  string `json:"state"` // active / established / closed / error
	Err    string `json:"err,omitempty"`
}

// Stats tracks live and recently finished connections. Everything lives in
// memory by design: the detail page wants "current connections", not history.
type Stats struct {
	mu       sync.Mutex
	nextID   uint64
	active   map[uint64]*connRecord
	recent   []*connRecord // finished, capped ring
	cap      int
	started  time.Time
	total    int64
	failures int64
}

type connRecord struct {
	info   ConnInfo
	closed bool
	endAt  time.Time
}

func NewStats() *Stats {
	return &Stats{
		active:  make(map[uint64]*connRecord),
		recent:  make([]*connRecord, 0, 100),
		cap:     100,
		started: time.Now(),
	}
}

// Open registers a new connection and returns its id plus a finish func.
func (s *Stats) Open(client, target, method string) (uint64, func(state, errMsg string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.total++
	rec := &connRecord{info: ConnInfo{
		ID:     s.nextID,
		Client: client,
		Target: target,
		Method: method,
		Since:  time.Now().UTC().Format(time.RFC3339),
		State:  "active",
	}}
	s.active[rec.info.ID] = rec
	id := rec.info.ID
	finish := func(state, errMsg string) {
		s.Finish(id, state, errMsg)
	}
	return id, finish
}

// Finish marks a connection ended and moves it to the recent ring.
func (s *Stats) Finish(id uint64, state, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.active[id]
	if !ok {
		return
	}
	delete(s.active, id)
	rec.closed = true
	rec.endAt = time.Now()
	rec.info.State = state
	rec.info.Err = errMsg
	s.recent = append(s.recent, rec)
	if len(s.recent) > s.cap {
		s.recent = s.recent[len(s.recent)-s.cap:]
	}
}

// AddBytes accumulates transferred bytes on a live connection.
func (s *Stats) AddBytes(id uint64, up, down int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.active[id]; ok {
		rec.info.Up += up
		rec.info.Down += down
	}
}

// CountFailures records one auth failure.
func (s *Stats) CountFailures() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures++
}

// Snapshot renders the active list (newest first) plus the recent ring.
func (s *Stats) Snapshot() (items []ConnInfo, active, connectActive, total, failures int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items = make([]ConnInfo, 0, len(s.active)+len(s.recent))
	now := time.Now()
	for _, rec := range s.active {
		ci := rec.info
		ci.Secs = int64(now.Sub(mustParse(ci.Since)).Seconds())
		if ci.Method == "CONNECT" && ci.State == "active" {
			ci.State = "established"
		}
		items = append(items, ci)
	}
	for i := len(s.recent) - 1; i >= 0 && len(items) < s.cap; i-- {
		ci := s.recent[i].info
		ci.Secs = int64(s.recent[i].endAt.Sub(mustParse(ci.Since)).Seconds())
		items = append(items, ci)
	}
	for _, rec := range s.active {
		if rec.info.Method == "CONNECT" {
			connectActive++
		}
	}
	return items, int64(len(s.active)), connectActive, s.total, s.failures
}

func mustParse(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now()
	}
	return t
}
