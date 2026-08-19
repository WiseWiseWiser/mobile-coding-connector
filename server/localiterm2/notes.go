package localiterm2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// NoteStore owns ~/.ai-critic/iterm-bookmarks.json (or a test path).
type NoteStore struct {
	path string
	mu   sync.Mutex
	doc  *NotesDocument
}

// NewNoteStore creates a store for the given JSON path. Load on first use.
func NewNoteStore(path string) *NoteStore {
	return &NoteStore{path: path}
}

// DefaultNotesPath returns ~/.ai-critic/iterm-bookmarks.json.
func DefaultNotesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", "iterm-bookmarks.json"), nil
}

// Path returns the backing file path.
func (s *NoteStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Document returns the in-memory document after Load/mutations (copy of map).
func (s *NoteStore) Document() *NotesDocument {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return emptyNotes()
	}
	out := cloneNotes(s.doc)
	normalizeNotesDoc(out)
	return out
}

func emptyNotes() *NotesDocument {
	return &NotesDocument{
		Version: NotesDocVersion,
		Items:   []*NoteItem{},
		Notes:   map[string]*NoteRecord{},
	}
}

func cloneNotes(doc *NotesDocument) *NotesDocument {
	if doc == nil {
		return emptyNotes()
	}
	out := &NotesDocument{
		Version: doc.Version,
		Items:   make([]*NoteItem, 0, len(doc.Items)),
		Notes:   make(map[string]*NoteRecord, len(doc.Notes)),
	}
	if out.Version == 0 {
		out.Version = NotesDocVersion
	}
	for _, it := range doc.Items {
		if it == nil {
			continue
		}
		out.Items = append(out.Items, cloneNoteItem(it))
	}
	for k, v := range doc.Notes {
		if v == nil {
			continue
		}
		out.Notes[k] = cloneNoteRecord(v)
	}
	return out
}

func cloneNoteItem(it *NoteItem) *NoteItem {
	if it == nil {
		return nil
	}
	cp := *it
	if it.LastSeen != nil {
		ls := *it.LastSeen
		cp.LastSeen = &ls
	}
	return &cp
}

func cloneNoteRecord(rec *NoteRecord) *NoteRecord {
	if rec == nil {
		return nil
	}
	cp := *rec
	if rec.LastSeen != nil {
		ls := *rec.LastSeen
		cp.LastSeen = &ls
	}
	return &cp
}

type notesFile struct {
	Version int                    `json:"version"`
	Items   []*NoteItem            `json:"items"`
	Notes   map[string]*NoteRecord `json:"notes"`
}

// UnmarshalJSON accepts v1 {notes:{uuid:record}} and v2 {items:[...]}.
func (d *NotesDocument) UnmarshalJSON(data []byte) error {
	var raw notesFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.Version = raw.Version
	d.Items = raw.Items
	d.Notes = raw.Notes
	normalizeNotesDoc(d)
	return nil
}

// MarshalJSON emits version 2 with a flat items list.
func (d NotesDocument) MarshalJSON() ([]byte, error) {
	items := d.Items
	if len(items) == 0 && len(d.Notes) > 0 {
		items = notesMapToItems(d.Notes)
	}
	if items == nil {
		items = []*NoteItem{}
	}
	return json.Marshal(struct {
		Version int         `json:"version"`
		Items   []*NoteItem `json:"items"`
	}{Version: 2, Items: items})
}

func normalizeNotesDoc(d *NotesDocument) {
	if d == nil {
		return
	}
	if d.Items == nil {
		d.Items = []*NoteItem{}
	}
	if len(d.Items) == 0 && len(d.Notes) > 0 {
		d.Items = notesMapToItems(d.Notes)
	}
	d.Version = NotesDocVersion
	syncNotesMap(d)
}

func notesItems(doc *NotesDocument) []*NoteItem {
	if doc == nil {
		return nil
	}
	if len(doc.Items) > 0 {
		return doc.Items
	}
	if len(doc.Notes) == 0 {
		return nil
	}
	return notesMapToItems(doc.Notes)
}

func notesMapToItems(notes map[string]*NoteRecord) []*NoteItem {
	if len(notes) == 0 {
		return nil
	}
	keys := make([]string, 0, len(notes))
	for k := range notes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*NoteItem, 0, len(keys))
	for _, k := range keys {
		rec := notes[k]
		if rec == nil {
			continue
		}
		out = append(out, noteRecordToItem(k, rec))
	}
	return out
}

func noteRecordToItem(itermID string, rec *NoteRecord) *NoteItem {
	it := &NoteItem{
		Note:           rec.Note,
		Bookmarked:     rec.Bookmarked,
		UpdatedAt:      rec.UpdatedAt,
		AgentRunner:    rec.AgentRunner,
		GrokSessionID:  rec.GrokSessionID,
		ITermSessionID: strings.TrimSpace(itermID),
	}
	if id := strings.TrimSpace(rec.ITermSessionID); id != "" {
		it.ITermSessionID = id
	}
	if rec.LastSeen != nil {
		ls := *rec.LastSeen
		it.LastSeen = &ls
	}
	return it
}

func itemToNoteRecord(it *NoteItem) *NoteRecord {
	rec := &NoteRecord{
		Note:           it.Note,
		Bookmarked:     it.Bookmarked,
		UpdatedAt:      it.UpdatedAt,
		AgentRunner:    it.AgentRunner,
		GrokSessionID:  it.GrokSessionID,
		ITermSessionID: it.ITermSessionID,
	}
	if it.LastSeen != nil {
		ls := *it.LastSeen
		rec.LastSeen = &ls
	}
	return rec
}

func syncNotesMap(d *NotesDocument) {
	d.Notes = make(map[string]*NoteRecord, len(d.Items))
	for _, it := range d.Items {
		if it == nil {
			continue
		}
		id := strings.TrimSpace(it.ITermSessionID)
		if id == "" {
			continue
		}
		d.Notes[id] = itemToNoteRecord(it)
	}
}

func findItemByITerm(items []*NoteItem, sessionID string) int {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return -1
	}
	for i, it := range items {
		if it != nil && strings.TrimSpace(it.ITermSessionID) == sessionID {
			return i
		}
	}
	return -1
}

type itemIdentity struct {
	AgentRunner   string
	GrokSessionID string
}

func (s *NoteStore) loadLocked() error {
	if s.doc != nil {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.doc = emptyNotes()
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		s.doc = emptyNotes()
		return nil
	}
	var doc NotesDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse iterm-bookmarks.json: %w", err)
	}
	normalizeNotesDoc(&doc)
	s.doc = &doc
	return nil
}

func (s *NoteStore) saveLocked() error {
	if s.doc == nil {
		s.doc = emptyNotes()
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

// Put upserts a note. Empty/whitespace note deletes the entry unless it stays bookmarked.
// now is used for updated_at; lastSeen may be nil.
func (s *NoteStore) Put(sessionID, note string, now time.Time, lastSeen *NoteLastSeen) error {
	n := note
	return s.Update(sessionID, &n, nil, now, lastSeen)
}

// Update patches note and/or bookmarked. Nil pointers leave that field unchanged.
// The record is deleted only when the result is an empty note and not bookmarked.
func (s *NoteStore) Update(sessionID string, note *string, bookmarked *bool, now time.Time, lastSeen *NoteLastSeen) error {
	return s.update(sessionID, note, bookmarked, now, lastSeen, nil)
}

func (s *NoteStore) update(sessionID string, note *string, bookmarked *bool, now time.Time, lastSeen *NoteLastSeen, ident *itemIdentity) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	normalizeNotesDoc(s.doc)
	it := &NoteItem{ITermSessionID: sessionID}
	idx := findItemByITerm(s.doc.Items, sessionID)
	if idx >= 0 {
		it = cloneNoteItem(s.doc.Items[idx])
		it.ITermSessionID = sessionID
	}
	if note != nil {
		it.Note = strings.TrimSpace(*note)
	}
	if bookmarked != nil {
		it.Bookmarked = *bookmarked
	}
	if lastSeen != nil {
		ls := *lastSeen
		it.LastSeen = &ls
	}
	if ident != nil {
		it.AgentRunner = strings.TrimSpace(ident.AgentRunner)
		it.GrokSessionID = strings.TrimSpace(ident.GrokSessionID)
	}
	if it.Note == "" && !it.Bookmarked {
		if idx >= 0 {
			s.doc.Items = append(s.doc.Items[:idx], s.doc.Items[idx+1:]...)
		}
		syncNotesMap(s.doc)
		return s.saveLocked()
	}
	if now.IsZero() {
		now = time.Now()
	}
	it.UpdatedAt = now.UTC().Format(time.RFC3339)
	if idx >= 0 {
		s.doc.Items[idx] = it
	} else {
		s.doc.Items = append(s.doc.Items, it)
	}
	s.doc.Version = NotesDocVersion
	syncNotesMap(s.doc)
	return s.saveLocked()
}

// Get returns the record for sessionID, or nil.
func (s *NoteStore) Get(sessionID string) *NoteRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return nil
	}
	normalizeNotesDoc(s.doc)
	rec := s.doc.Notes[strings.TrimSpace(sessionID)]
	if rec == nil {
		return nil
	}
	return cloneNoteRecord(rec)
}
