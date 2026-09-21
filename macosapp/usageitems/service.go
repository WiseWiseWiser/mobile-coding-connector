package usageitems

import (
	"fmt"
	"strings"
	"sync"
)

// AddRequest describes a new item. Empty ID derives from Label and an empty
// Label derives from the kind; Enabled defaults to true.
type AddRequest struct {
	Item         Item
	Enabled      *bool
	Default      bool
	SkipValidate bool
	Strict       bool
}

// UpdateRequest describes field changes for an existing item.
type UpdateRequest struct {
	ID           string
	Update       ItemUpdate
	SkipValidate bool
	Strict       bool
}

// RemoveResult reports a deletion and the resulting menu-bar default.
type RemoveResult struct {
	Removed  string
	Default  string
	Warnings []string
}

// Service owns the registry and one fetch worker per item.
type Service struct {
	store *Store

	mu      sync.Mutex
	workers map[string]*workerState
	started bool

	// factory builds provider workers; tests replace it via
	// TestExported_SetSnapshotFetcher.
	factory workerFactory
}

// workerFactory builds the fetch worker for one item.
type workerFactory func(Item) (worker, error)

type workerState struct {
	item Item
	w    worker
}

// NewService creates a usage item service backed by store.
func NewService(store *Store) *Service {
	return &Service{store: store, workers: make(map[string]*workerState)}
}

// Start begins background refresh loops for every registered item.
func (s *Service) Start() {
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	reg, err := s.store.Load()
	if err != nil {
		return
	}
	s.sync(reg)
}

// Stop ends background refresh loops.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, ws := range s.workers {
		ws.w.Stop()
		delete(s.workers, id)
	}
	s.started = false
}

// Registry returns the persisted registry.
func (s *Service) Registry() (*Registry, error) {
	return s.store.Load()
}

// List returns every item with rendered menu text. When refresh is true, each
// enabled worker fetches fresh usage from its provider before rendering;
// otherwise the last cached snapshot is rendered.
func (s *Service) List(refresh bool) (*ListResponse, error) {
	reg, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	s.sync(reg)
	if refresh {
		s.refresh(reg)
	}
	return s.render(reg, refresh), nil
}

// refresh runs a synchronous provider fetch for every enabled worker, so the
// rendered view reflects current usage rather than the cached snapshot.
func (s *Service) refresh(reg *Registry) {
	for _, it := range reg.Items {
		if !it.Enabled {
			continue
		}
		ws := s.worker(it.ID)
		if ws != nil {
			ws.w.FetchNow()
		}
	}
}

// Show returns one item with rendered menu text, fetching when stale.
func (s *Service) Show(id string) (*ItemView, error) {
	reg, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	it := reg.Find(id)
	if it == nil {
		return nil, notFoundf("unknown usage item %q; known items: %s", id, reg.ids())
	}
	s.sync(reg)
	ws := s.worker(it.ID)
	if ws != nil {
		ws.w.EnsureFetch()
	}
	view := itemView(*it, ws)
	return &view, nil
}

// Add registers a new item, optionally validating it against the provider.
func (s *Service) Add(req AddRequest) (*ItemView, []string, error) {
	reg, err := s.store.Load()
	if err != nil {
		return nil, nil, err
	}

	it := req.Item
	it.ID = strings.TrimSpace(it.ID)
	it.Label = strings.TrimSpace(it.Label)
	it.Kind = strings.TrimSpace(it.Kind)
	it.Home = strings.TrimSpace(it.Home)
	it.APIURL = strings.TrimSpace(it.APIURL)
	if it.ID == "" {
		it.ID = Slug(it.Label)
	}
	if it.ID == "" {
		return nil, nil, invalidf("usage item: --id or --label is required")
	}
	if it.Label == "" {
		it.Label = DefaultLabel(it.Kind, it.ID, reg.Items)
	}
	if req.Enabled == nil {
		it.Enabled = true
	} else {
		it.Enabled = *req.Enabled
	}
	if err := reg.Add(it); err != nil {
		return nil, nil, err
	}

	w, err := s.buildWorker(it)
	if err != nil {
		return nil, nil, err
	}

	var warnings []string
	if !req.SkipValidate {
		w.FetchNow()
		if snap := w.Snapshot(); snap.Status != StatusReady {
			msg := fmt.Sprintf("%s fetch failed: %s", it.ID, fetchFailure(snap))
			if req.Strict {
				w.Stop()
				return nil, nil, invalidf("%s", msg)
			}
			warnings = append(warnings, msg)
		}
	}

	if req.Default {
		if err := reg.SetDefault(it.ID, false); err != nil {
			return nil, nil, err
		}
	}
	if err := s.store.Save(reg); err != nil {
		w.Stop()
		return nil, nil, err
	}

	s.mu.Lock()
	if s.started {
		w.Start()
	}
	s.workers[it.ID] = &workerState{item: it, w: w}
	s.mu.Unlock()

	ws := &workerState{item: it, w: w}
	view := itemView(it, ws)
	return &view, warnings, nil
}

// Update changes an item, validating when the provider binding changes.
func (s *Service) Update(req UpdateRequest) (*ItemView, []string, error) {
	reg, err := s.store.Load()
	if err != nil {
		return nil, nil, err
	}
	before := reg.Find(req.ID)
	if before == nil {
		return nil, nil, notFoundf("unknown usage item %q; known items: %s", req.ID, reg.ids())
	}
	previous := *before
	it, err := reg.Update(req.ID, req.Update)
	if err != nil {
		return nil, nil, err
	}

	var warnings []string
	if rebuild := !sameWorker(previous, *it); rebuild && !req.SkipValidate {
		w, err := s.buildWorker(*it)
		if err != nil {
			return nil, nil, err
		}
		w.FetchNow()
		snap := w.Snapshot()
		w.Stop()
		if snap.Status != StatusReady {
			msg := fmt.Sprintf("%s fetch failed: %s", it.ID, fetchFailure(snap))
			if req.Strict {
				return nil, nil, invalidf("%s", msg)
			}
			warnings = append(warnings, msg)
		}
	}

	if err := s.store.Save(reg); err != nil {
		return nil, nil, err
	}
	s.sync(reg)
	ws := s.worker(it.ID)
	if ws != nil && it.Enabled {
		ws.w.EnsureFetch()
	}
	view := itemView(*it, ws)
	return &view, warnings, nil
}

// Remove deletes an item and its worker.
func (s *Service) Remove(id string) (*RemoveResult, error) {
	reg, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	removed, warnings, err := reg.Remove(id)
	if err != nil {
		return nil, err
	}
	if err := s.store.Save(reg); err != nil {
		return nil, err
	}
	s.sync(reg)
	return &RemoveResult{Removed: removed.ID, Default: reg.Default, Warnings: warnings}, nil
}

// SetDefault chooses the fixed menu-bar item, or rotation over enabled items.
func (s *Service) SetDefault(id string, rotate bool) (*ListResponse, error) {
	reg, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if err := reg.SetDefault(id, rotate); err != nil {
		return nil, err
	}
	if err := s.store.Save(reg); err != nil {
		return nil, err
	}
	s.sync(reg)
	return s.render(reg, false), nil
}

func (s *Service) render(reg *Registry, fresh bool) *ListResponse {
	out := &ListResponse{
		Version: reg.Version,
		Default: reg.Default,
		Rotate:  reg.Rotate,
		Items:   make([]ItemView, 0, len(reg.Items)),
	}
	for _, it := range reg.Items {
		ws := s.worker(it.ID)
		if ws != nil && it.Enabled && !fresh {
			ws.w.EnsureFetch()
		}
		out.Items = append(out.Items, itemView(it, ws))
	}
	return out
}

// sync reconciles workers with the registry: new kinds or provider bindings get
// a fresh worker, removed items get stopped.
func (s *Service) sync(reg *Registry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]bool, len(reg.Items))
	for _, it := range reg.Items {
		seen[it.ID] = true
		ws := s.workers[it.ID]
		if ws != nil && sameWorker(ws.item, it) {
			ws.item = it
			continue
		}
		if ws != nil {
			ws.w.Stop()
			delete(s.workers, it.ID)
		}
		w, err := s.buildWorker(it)
		if err != nil {
			continue
		}
		if s.started {
			w.Start()
		}
		s.workers[it.ID] = &workerState{item: it, w: w}
	}
	for id, ws := range s.workers {
		if !seen[id] {
			ws.w.Stop()
			delete(s.workers, id)
		}
	}
}

func (s *Service) buildWorker(it Item) (worker, error) {
	if s.factory != nil {
		return s.factory(it)
	}
	return newWorker(it)
}

// TestExported_SetSnapshotFetcher replaces provider fetching with an injected
// snapshot per item, leaving the registry and rendering untouched.
func TestExported_SetSnapshotFetcher(s *Service, fetch func(Item) (Snapshot, error)) {
	s.factory = func(it Item) (worker, error) {
		item := it
		return newFuncWorker(func() (Snapshot, error) { return fetch(item) }), nil
	}
}

func (s *Service) worker(id string) *workerState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.workers[id]
}

func itemView(it Item, ws *workerState) ItemView {
	view := ItemView{Item: it, Status: StatusLoading}
	if ws == nil {
		return view
	}
	snap := ws.w.Snapshot()
	view.Status = snap.Status
	view.Title = FormatTitle(it.Label, snap.Status, snap.Percent, snap.Error)
	view.Dropdown = FormatDropdown(it.Label, snap.Status, snap.Body, snap.Error)
	view.Detail = snap.Detail
	view.UsageURL = snap.UsageURL
	view.Error = snap.Error
	view.UpdatedAt = snap.UpdatedAt
	return view
}

func fetchFailure(snap Snapshot) string {
	if msg := strings.TrimSpace(snap.Error); msg != "" {
		return msg
	}
	if snap.Status != StatusReady {
		return "no usage data yet"
	}
	return ""
}
