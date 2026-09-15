// Package usageitems stores the usage sources shown in the macOS menu bar and
// renders each one's title and dropdown text. Items live on the server so a CLI
// can add, update, remove, and select them while the app stays a thin renderer.
package usageitems

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Item kinds. Each kind maps to one provider fetch.
const (
	KindGrok        = "grok"
	KindCodex       = "codex"
	KindCommandCode = "commandcode"
)

// Item is one registered usage source.
type Item struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Enabled bool   `json:"enabled"`
	// Home is the provider config dir; required for every kind except grok and
	// codex, which read the machine's default account.
	Home   string `json:"home,omitempty"`
	APIURL string `json:"api_url,omitempty"`
}

// ItemUpdate carries optional field changes; nil leaves a field untouched.
type ItemUpdate struct {
	Label   *string
	Kind    *string
	Home    *string
	APIURL  *string
	Enabled *bool
}

// Registry is the persisted usage item list plus menu-bar selection state.
type Registry struct {
	Version int    `json:"version"`
	Default string `json:"default"`
	Rotate  bool   `json:"rotate"`
	Items   []Item `json:"items"`
}

// RegistryVersion is the schema version written to new registries.
const RegistryVersion = 1

// ValidKinds lists the supported item kinds in display order.
func ValidKinds() []string {
	return []string{KindGrok, KindCodex, KindCommandCode}
}

// ProviderLabel is the default menu label for a kind.
func ProviderLabel(kind string) string {
	switch kind {
	case KindGrok:
		return "Grok"
	case KindCodex:
		return "Codex"
	case KindCommandCode:
		return "CommandCode"
	default:
		return kind
	}
}

// Seed is the registry used before any item is configured: today's behavior.
func Seed() *Registry {
	return &Registry{
		Version: RegistryVersion,
		Default: KindGrok,
		Rotate:  true,
		Items: []Item{
			{ID: KindGrok, Label: ProviderLabel(KindGrok), Kind: KindGrok, Enabled: true},
			{ID: KindCodex, Label: ProviderLabel(KindCodex), Kind: KindCodex, Enabled: true},
		},
	}
}

// Slug converts a label into a stable item id ("CC v1" -> "cc-v1").
func Slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// ValidateItem checks the fields a kind requires.
func ValidateItem(it Item) error {
	if strings.TrimSpace(it.ID) == "" {
		return errors.New("id is required")
	}
	if strings.TrimSpace(it.Label) == "" {
		return errors.New("label is required")
	}
	switch it.Kind {
	case KindCommandCode:
		if strings.TrimSpace(it.Home) == "" {
			return fmt.Errorf("--home is required for kind %s", KindCommandCode)
		}
	case KindGrok, KindCodex:
	default:
		return fmt.Errorf("unknown kind %q; known kinds: %s", it.Kind, strings.Join(ValidKinds(), ", "))
	}
	return nil
}

// DefaultLabel returns the label to use when --label is omitted: the provider
// name, suffixed with the id once that name is already taken.
func DefaultLabel(kind, id string, existing []Item) string {
	base := ProviderLabel(kind)
	for _, it := range existing {
		if strings.EqualFold(it.Label, base) {
			return base + " " + id
		}
	}
	return base
}

// Find returns the item with the given id, or nil.
func (r *Registry) Find(id string) *Item {
	for i := range r.Items {
		if r.Items[i].ID == id {
			return &r.Items[i]
		}
	}
	return nil
}

// EnabledItems returns enabled items in registry order.
func (r *Registry) EnabledItems() []Item {
	var out []Item
	for _, it := range r.Items {
		if it.Enabled {
			out = append(out, it)
		}
	}
	return out
}

// Add appends an item, normalizing its fields.
func (r *Registry) Add(it Item) error {
	it.ID = strings.TrimSpace(it.ID)
	it.Kind = strings.TrimSpace(it.Kind)
	it.Label = strings.TrimSpace(it.Label)
	it.Home = strings.TrimSpace(it.Home)
	it.APIURL = strings.TrimSpace(it.APIURL)
	if r.Find(it.ID) != nil {
		return conflictf("usage item %q already exists; run 'update %s' instead", it.ID, it.ID)
	}
	if err := ValidateItem(it); err != nil {
		return invalidf("usage item %s: %v", it.ID, err)
	}
	r.Items = append(r.Items, it)
	if strings.TrimSpace(r.Default) == "" {
		r.Default = it.ID
	}
	return nil
}

// Update applies non-nil field changes to the item with the given id.
func (r *Registry) Update(id string, upd ItemUpdate) (*Item, error) {
	it := r.Find(id)
	if it == nil {
		return nil, notFoundf("unknown usage item %q; known items: %s", id, r.ids())
	}
	updated := *it
	if upd.Label != nil {
		updated.Label = strings.TrimSpace(*upd.Label)
	}
	if upd.Kind != nil {
		updated.Kind = strings.TrimSpace(*upd.Kind)
	}
	if upd.Home != nil {
		updated.Home = strings.TrimSpace(*upd.Home)
	}
	if upd.APIURL != nil {
		updated.APIURL = strings.TrimSpace(*upd.APIURL)
	}
	if upd.Enabled != nil {
		updated.Enabled = *upd.Enabled
	}
	if err := ValidateItem(updated); err != nil {
		return nil, invalidf("usage item %s: %v", id, err)
	}
	*it = updated
	return it, nil
}

// Remove drops an item and reports the new menu-bar default plus any warnings.
func (r *Registry) Remove(id string) (Item, []string, error) {
	it := r.Find(id)
	if it == nil {
		return Item{}, nil, notFoundf("unknown usage item %q; known items: %s", id, r.ids())
	}
	removed := *it
	kept := make([]Item, 0, len(r.Items))
	for _, item := range r.Items {
		if item.ID != removed.ID {
			kept = append(kept, item)
		}
	}
	r.Items = kept

	var warnings []string
	if r.Default == removed.ID {
		previous := r.Default
		r.Default = defaultAfterRemoval(r.Items, r.Default)
		warnings = append(warnings, fmt.Sprintf("%s was the menu bar default; default is now %s", previous, displayDefault(r.Default)))
	}
	return removed, warnings, nil
}

func defaultAfterRemoval(items []Item, skip string) string {
	for _, it := range items {
		if it.Enabled && it.ID != skip {
			return it.ID
		}
	}
	for _, it := range items {
		if it.ID != skip {
			return it.ID
		}
	}
	return ""
}

// SetDefault selects the fixed menu-bar item, or enables rotation over the
// enabled items when rotate is true. An empty id keeps the current item.
func (r *Registry) SetDefault(id string, rotate bool) error {
	id = strings.TrimSpace(id)
	if id != "" && r.Find(id) == nil {
		return notFoundf("unknown usage item %q; known items: %s", id, r.ids())
	}
	if id != "" {
		r.Default = id
	}
	r.Rotate = rotate
	if !rotate && strings.TrimSpace(r.Default) == "" {
		if enabled := r.EnabledItems(); len(enabled) > 0 {
			r.Default = enabled[0].ID
		}
	}
	return nil
}

// Normalize repairs a registry read from disk: schema version, item ids, labels,
// and a default that points at an existing item.
func (r *Registry) Normalize() {
	if r.Version <= 0 {
		r.Version = RegistryVersion
	}
	items := make([]Item, 0, len(r.Items))
	seen := make(map[string]bool, len(r.Items))
	for _, it := range r.Items {
		it.ID = strings.TrimSpace(it.ID)
		it.Kind = strings.TrimSpace(it.Kind)
		it.Label = strings.TrimSpace(it.Label)
		it.Home = strings.TrimSpace(it.Home)
		it.APIURL = strings.TrimSpace(it.APIURL)
		if it.ID == "" || seen[it.ID] {
			continue
		}
		if it.Label == "" {
			it.Label = ProviderLabel(it.Kind)
		}
		seen[it.ID] = true
		items = append(items, it)
	}
	r.Items = items
	if r.Default == "" || !seen[r.Default] {
		r.Default = defaultAfterRemoval(r.Items, "")
	}
}

func (r *Registry) ids() string {
	if len(r.Items) == 0 {
		return "(none)"
	}
	ids := make([]string, 0, len(r.Items))
	for _, it := range r.Items {
		ids = append(ids, it.ID)
	}
	return strings.Join(ids, ", ")
}

func displayDefault(id string) string {
	if id == "" {
		return "(none)"
	}
	return id
}

// Store is a file-backed registry store, one JSON file.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore creates a store at path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Path returns the backing file path.
func (s *Store) Path() string { return s.path }

// Load reads the registry, seeding grok and codex when the file does not exist.
func (s *Store) Load() (*Registry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store) loadLocked() (*Registry, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read usage items: %w", err)
		}
		reg := Seed()
		if err := s.saveLocked(reg); err != nil {
			return nil, err
		}
		return reg, nil
	}
	var reg Registry
	if err := json.Unmarshal(raw, &reg); err != nil {
		return nil, fmt.Errorf("parse usage items %s: %w", s.path, err)
	}
	reg.Normalize()
	return &reg, nil
}

// Save writes the registry to disk.
func (s *Store) Save(reg *Registry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(reg)
}

func (s *Store) saveLocked(reg *Registry) error {
	reg.Normalize()
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal usage items: %w", err)
	}
	if dir := filepath.Dir(s.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create usage items dir: %w", err)
		}
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("write usage items %s: %w", s.path, err)
	}
	return nil
}

// Update loads, mutates, and saves the registry in one critical section.
func (s *Store) Update(fn func(*Registry) error) (*Registry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reg, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	if err := fn(reg); err != nil {
		return nil, err
	}
	if err := s.saveLocked(reg); err != nil {
		return nil, err
	}
	return reg, nil
}

// SortedIDs is used by error messages and tests.
func (r *Registry) SortedIDs() []string {
	ids := make([]string, 0, len(r.Items))
	for _, it := range r.Items {
		ids = append(ids, it.ID)
	}
	sort.Strings(ids)
	return ids
}
