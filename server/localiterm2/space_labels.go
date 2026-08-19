package localiterm2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const spaceLabelsDocVersion = 1

// SpaceRef is one live type-0 Desktop (dense index + CGS identity).
type SpaceRef struct {
	SpaceIndex int
	SpaceID    uint64
	UUID       string
	Current    bool
}

// SpaceLabel is one persisted Space name.
type SpaceLabel struct {
	SpaceID   uint64 `json:"space_id"`
	UUID      string `json:"uuid,omitempty"`
	Label     string `json:"label"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// SpaceLabelsDocument is the versioned file at ~/.ai-critic/space-labels.json.
type SpaceLabelsDocument struct {
	Version int           `json:"version"`
	Labels  []*SpaceLabel `json:"labels"`
}

// SpaceLabelStore owns ~/.ai-critic/space-labels.json (or a test path).
type SpaceLabelStore struct {
	path string
	mu   sync.Mutex
	doc  *SpaceLabelsDocument
}

// NewSpaceLabelStore creates a store for the given JSON path. Load on first use.
func NewSpaceLabelStore(path string) *SpaceLabelStore {
	return &SpaceLabelStore{path: path}
}

// DefaultSpaceLabelsPath returns ~/.ai-critic/space-labels.json.
func DefaultSpaceLabelsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", "space-labels.json"), nil
}

// Path returns the backing file path.
func (s *SpaceLabelStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func emptySpaceLabels() *SpaceLabelsDocument {
	return &SpaceLabelsDocument{
		Version: spaceLabelsDocVersion,
		Labels:  []*SpaceLabel{},
	}
}

func cloneSpaceLabels(doc *SpaceLabelsDocument) *SpaceLabelsDocument {
	if doc == nil {
		return emptySpaceLabels()
	}
	out := &SpaceLabelsDocument{
		Version: spaceLabelsDocVersion,
		Labels:  make([]*SpaceLabel, 0, len(doc.Labels)),
	}
	for _, it := range doc.Labels {
		if it == nil {
			continue
		}
		cp := *it
		out.Labels = append(out.Labels, &cp)
	}
	return out
}

// Document returns a copy of the in-memory document after Load.
func (s *SpaceLabelStore) Document() *SpaceLabelsDocument {
	if s == nil {
		return emptySpaceLabels()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return emptySpaceLabels()
	}
	return cloneSpaceLabels(s.doc)
}

func (s *SpaceLabelStore) loadLocked() error {
	if s.doc != nil {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.doc = emptySpaceLabels()
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		s.doc = emptySpaceLabels()
		return nil
	}
	var doc SpaceLabelsDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse space-labels.json: %w", err)
	}
	if doc.Labels == nil {
		doc.Labels = []*SpaceLabel{}
	}
	doc.Version = spaceLabelsDocVersion
	s.doc = &doc
	return nil
}

func (s *SpaceLabelStore) saveLocked() error {
	if s.doc == nil {
		s.doc = emptySpaceLabels()
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Set upserts a label. Empty/whitespace label deletes the row.
func (s *SpaceLabelStore) Set(spaceID uint64, uuid, label string, now time.Time) error {
	if s == nil {
		return fmt.Errorf("space label store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	uuid = strings.TrimSpace(uuid)
	label = strings.TrimSpace(label)
	if spaceID == 0 && uuid == "" {
		return fmt.Errorf("space_id is required")
	}
	if label == "" {
		s.doc.Labels = removeSpaceLabel(s.doc.Labels, spaceID, uuid)
		return s.saveLocked()
	}
	idx := findSpaceLabel(s.doc.Labels, spaceID, uuid)
	rec := &SpaceLabel{
		SpaceID:   spaceID,
		UUID:      uuid,
		Label:     label,
		UpdatedAt: now.UTC().Format(time.RFC3339),
	}
	if idx >= 0 {
		if rec.SpaceID == 0 {
			rec.SpaceID = s.doc.Labels[idx].SpaceID
		}
		if rec.UUID == "" {
			rec.UUID = s.doc.Labels[idx].UUID
		}
		s.doc.Labels[idx] = rec
	} else {
		s.doc.Labels = append(s.doc.Labels, rec)
	}
	return s.saveLocked()
}

// Reconcile drops labels whose space_id and uuid are both missing from live,
// and rewrites space_id when a uuid rematch finds a new id. Returns the next doc.
func (s *SpaceLabelStore) Reconcile(live []SpaceRef) *SpaceLabelsDocument {
	if s == nil {
		return emptySpaceLabels()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return emptySpaceLabels()
	}
	next, changed := reconcileSpaceLabels(s.doc.Labels, live)
	if changed {
		s.doc.Labels = next
		_ = s.saveLocked()
	}
	return cloneSpaceLabels(s.doc)
}

func findSpaceLabel(labels []*SpaceLabel, spaceID uint64, uuid string) int {
	uuid = strings.TrimSpace(uuid)
	for i, it := range labels {
		if it == nil {
			continue
		}
		if spaceID != 0 && it.SpaceID == spaceID {
			return i
		}
	}
	if uuid == "" {
		return -1
	}
	for i, it := range labels {
		if it == nil {
			continue
		}
		if strings.TrimSpace(it.UUID) == uuid {
			return i
		}
	}
	return -1
}

func removeSpaceLabel(labels []*SpaceLabel, spaceID uint64, uuid string) []*SpaceLabel {
	idx := findSpaceLabel(labels, spaceID, uuid)
	if idx < 0 {
		return labels
	}
	return append(labels[:idx], labels[idx+1:]...)
}

// reconcileSpaceLabels is the pure keep/drop/relink step.
func reconcileSpaceLabels(stored []*SpaceLabel, live []SpaceRef) ([]*SpaceLabel, bool) {
	byID := map[uint64]SpaceRef{}
	byUUID := map[string]SpaceRef{}
	for _, s := range live {
		if s.SpaceID != 0 {
			byID[s.SpaceID] = s
		}
		if u := strings.TrimSpace(s.UUID); u != "" {
			byUUID[u] = s
		}
	}
	out := make([]*SpaceLabel, 0, len(stored))
	changed := false
	for _, it := range stored {
		if it == nil {
			changed = true
			continue
		}
		if hit, ok := byID[it.SpaceID]; ok && it.SpaceID != 0 {
			cp := *it
			if u := strings.TrimSpace(hit.UUID); u != "" && cp.UUID != u {
				cp.UUID = u
				changed = true
			}
			out = append(out, &cp)
			continue
		}
		if u := strings.TrimSpace(it.UUID); u != "" {
			if hit, ok := byUUID[u]; ok {
				cp := *it
				if cp.SpaceID != hit.SpaceID {
					cp.SpaceID = hit.SpaceID
					changed = true
				}
				out = append(out, &cp)
				continue
			}
		}
		changed = true
	}
	if !changed && len(out) == len(stored) {
		return stored, false
	}
	return out, true
}

// ApplySpaceLabels stamps live CGS ids onto Desktop groups and joins labels.
// It does not add or drop headings — ClipDesktops owns that.
// When live is empty, existing SpaceID/UUID on groups are used for the join.
func ApplySpaceLabels(inv Inventory, live []SpaceRef, doc *SpaceLabelsDocument) Inventory {
	if doc == nil {
		doc = emptySpaceLabels()
	}
	byIndex := map[int]SpaceRef{}
	for _, s := range live {
		byIndex[s.SpaceIndex] = s
	}
	desktops := make([]DesktopGroup, len(inv.Desktops))
	for i, g := range inv.Desktops {
		g.Current = false
		if ref, ok := byIndex[g.SpaceIndex]; ok {
			if ref.SpaceID != 0 {
				g.SpaceID = ref.SpaceID
			}
			if u := strings.TrimSpace(ref.UUID); u != "" {
				g.SpaceUUID = u
			}
			g.Current = ref.Current
		}
		g.Label = lookupSpaceLabel(doc.Labels, g.SpaceID, g.SpaceUUID)
		desktops[i] = g
	}
	inv.Desktops = desktops
	return inv
}

func lookupSpaceLabel(labels []*SpaceLabel, spaceID uint64, uuid string) string {
	idx := findSpaceLabel(labels, spaceID, uuid)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(labels[idx].Label)
}
