package eventbus

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
)

// Hub is an in-memory event bus with a fixed-capacity ring and live fan-out.
type Hub struct {
	mu          sync.Mutex
	ring        []sharedeb.Event
	ringSize    int
	nextID      int // next write index when ring is full (circular)
	count       int // number of valid events in ring (≤ ringSize)
	subscribers map[uint64]chan sharedeb.Event
	subSeq      uint64
}

// NewHub creates a Hub that retains up to ringSize recent events.
// If ringSize <= 0, a default of 200 is used.
func NewHub(ringSize int) *Hub {
	if ringSize <= 0 {
		ringSize = 200
	}
	return &Hub{
		ring:        make([]sharedeb.Event, ringSize),
		ringSize:    ringSize,
		subscribers: make(map[uint64]chan sharedeb.Event),
	}
}

// Publish enriches missing id/ts, stores the event in the ring, and fans out
// to live subscribers. Returns the stored (possibly enriched) event.
func (h *Hub) Publish(ev sharedeb.Event) sharedeb.Event {
	if h == nil {
		return ev
	}
	if ev.ID == "" {
		ev.ID = newEventID()
	}
	if ev.TS == "" {
		ev.TS = time.Now().UTC().Format(time.RFC3339Nano)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.count < h.ringSize {
		h.ring[h.count] = ev
		h.count++
	} else {
		// Overwrite oldest; nextID is the write head when full.
		h.ring[h.nextID] = ev
		h.nextID = (h.nextID + 1) % h.ringSize
	}

	for _, ch := range h.subscribers {
		// Non-blocking send: drop if subscriber is slow.
		select {
		case ch <- ev:
		default:
		}
	}
	return ev
}

// Subscribe registers a live subscriber. cancel removes it and closes the channel.
func (h *Hub) Subscribe() (<-chan sharedeb.Event, func()) {
	if h == nil {
		ch := make(chan sharedeb.Event)
		close(ch)
		return ch, func() {}
	}
	ch := make(chan sharedeb.Event, 64)
	h.mu.Lock()
	h.subSeq++
	id := h.subSeq
	h.subscribers[id] = ch
	h.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			if existing, ok := h.subscribers[id]; ok {
				delete(h.subscribers, id)
				close(existing)
			}
			h.mu.Unlock()
		})
	}
	return ch, cancel
}

// Recent returns up to n most recent events in chronological order (oldest first).
// If n <= 0 or n > stored count, all stored events are returned.
func (h *Hub) Recent(n int) []sharedeb.Event {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	total := h.count
	if total == 0 {
		return nil
	}
	if n <= 0 || n > total {
		n = total
	}

	out := make([]sharedeb.Event, n)
	// Oldest retained index when ring is full is nextID; when not full, 0.
	start := 0
	if h.count == h.ringSize {
		start = h.nextID
	}
	// Skip (total - n) oldest to keep the newest n.
	skip := total - n
	for i := 0; i < n; i++ {
		idx := (start + skip + i) % h.ringSize
		out[i] = h.ring[idx]
	}
	return out
}

func newEventID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback: time-based id if entropy fails.
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b[:])
}
