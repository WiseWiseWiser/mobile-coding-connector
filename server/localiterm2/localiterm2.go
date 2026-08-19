// Package localiterm2 serves local-only iTerm2 APIs for the macOS menu-bar app:
// open a directory, inventory live sessions, focus a session, and persist notes.
package localiterm2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	spacelib "github.com/xhd2015/dot-pkgs/go-pkgs/computer-use/macos/space"
	"github.com/xhd2015/dot-pkgs/go-pkgs/libmark"
	shelliterm "github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
	kooliterm "github.com/xhd2015/kool/tools/iterm2"
)

// Path is the fixed product endpoint for local iTerm2 open.
const Path = "/api/local/iterm2/open"

// openRequest is the JSON body for Path.
type openRequest struct {
	Dir  string   `json:"dir"`
	Mode string   `json:"mode"`
	Send []string `json:"send"`
}

// Handler serves local iTerm2 endpoints. Hooks are injectible for tests;
// nil fields use production kool / dot-pkgs implementations.
type Handler struct {
	Open    func(dir string, cfg *shelliterm.Config) error
	Capture func() (*kooliterm.Snapshot, error)
	// CaptureSpace deep-captures windows on one 0-based Desktop. Nil uses
	// CaptureSnapshotWith SpaceAllow, or Capture when tests inject only that.
	CaptureSpace func(spaceIndex int) (*kooliterm.Snapshot, error)
	// CaptureStream yields each window as it is collected (tests / production stream).
	CaptureStream func(onWindow func(kooliterm.SnapshotWindow) error) (*kooliterm.Snapshot, error)
	// Layout is an IDs-only probe (windows/tabs/session IDs, no enrich).
	// Nil uses kool ListWindows + ListTabsAndSessions. Not a deep Capture.
	Layout func() (*kooliterm.Snapshot, error)
	// CountDesktops is the silent Desktop count (plist / CGS). Never open Mission Control.
	CountDesktops  func() (int, error)
	SpaceForWindow func(windowID uint64) (int, error)
	// ListSpaces lists live type-0 Desktops with CGS ids. Nil uses spacelib.ListUserSpaces.
	ListSpaces func() ([]SpaceRef, error)
	Focus      func(ref shelliterm.SessionRef) error
	Switch     func(desktop int) error
	// SwitchSpace activates a Space by CGS id64. Nil uses live SkyLight in this process.
	SwitchSpace  func(spaceID uint64) error
	ITermRunning func() bool
	Notes        *NoteStore
	SpaceLabels  *SpaceLabelStore
	Now          func() time.Time
	// CachePath is the durable last-good inventory JSON path.
	// Empty uses DefaultInventoryCachePath (~/.ai-critic/iterm-inventory-cache.json).
	CachePath string
	// ResolveMark returns live mark content for a PID. Nil uses libmark.Resolve.
	// Miss/error is empty; tests inject so inventory never reads ~/.mark.
	ResolveMark func(pid int) (string, error)

	cache inventoryCache
}

var (
	defaultNotes           *NoteStore
	defaultNotesOnce       sync.Once
	defaultSpaceLabels     *SpaceLabelStore
	defaultSpaceLabelsOnce sync.Once
)

// ParseOpenMode maps JSON mode strings to iterm2.OpenMode.
// Empty / "reuse" → ModeReuseCurrent; "new" → ModeForceNew; "smart" → ModeSmart.
// Unknown values return an error (do not fall through to ModeSmart zero-value).
func ParseOpenMode(s string) (shelliterm.OpenMode, error) {
	switch strings.TrimSpace(s) {
	case "", "reuse":
		return shelliterm.ModeReuseCurrent, nil
	case "new":
		return shelliterm.ModeForceNew, nil
	case "smart":
		return shelliterm.ModeSmart, nil
	default:
		return 0, fmt.Errorf("invalid open mode %q (want reuse, new, or smart)", s)
	}
}

// Register mounts open / inventory / focus / notes on mux.
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.Handle(Path, h)
	mux.HandleFunc(InventoryPath, h.handleInventory)
	mux.HandleFunc(InventoryStreamPath, h.handleInventoryStream)
	mux.HandleFunc(FocusPath, h.handleFocus)
	mux.HandleFunc(NotesPath, h.handleNotes)
	mux.HandleFunc(SpaceLabelsPath, h.handleSpaceLabels)
	mux.HandleFunc(SwitchSpacePath, h.handleSwitchSpace)
}

func (h *Handler) openFunc() func(dir string, cfg *shelliterm.Config) error {
	if h != nil && h.Open != nil {
		return h.Open
	}
	return shelliterm.OpenConfig
}

func (h *Handler) captureFn() func() (*kooliterm.Snapshot, error) {
	if h != nil && h.Capture != nil {
		return h.Capture
	}
	return func() (*kooliterm.Snapshot, error) {
		snap, _, err := kooliterm.CaptureSnapshotWith(h.captureOpts())
		return snap, err
	}
}

func (h *Handler) layoutFn() func() (*kooliterm.Snapshot, error) {
	if h != nil && h.Layout != nil {
		return h.Layout
	}
	return h.defaultLayout
}

// DefaultLayout lists windows/tabs/session IDs only (no process/agent enrich).
// Used by in-process CLI when no Layout inject is installed.
func (h *Handler) DefaultLayout() (*kooliterm.Snapshot, error) {
	return h.defaultLayout()
}

// defaultLayout lists windows/tabs/session IDs only (no process/agent enrich).
func (h *Handler) defaultLayout() (*kooliterm.Snapshot, error) {
	c := &kooliterm.SnapshotCollector{}
	headers, _, err := c.ListWindows()
	if err != nil {
		return nil, err
	}
	windows := make([]kooliterm.SnapshotWindow, 0, len(headers))
	for _, hdr := range headers {
		tabs, _, tabErr := c.ListTabsAndSessions(hdr.Index)
		if tabErr != nil {
			return nil, tabErr
		}
		hdr.Tabs = tabs
		windows = append(windows, hdr)
	}
	return &kooliterm.Snapshot{Windows: windows}, nil
}

func (h *Handler) countDesktopsFn() func() (int, error) {
	if h != nil && h.CountDesktops != nil {
		return h.CountDesktops
	}
	return func() (int, error) {
		return spacelib.CountDesktops()
	}
}

func (h *Handler) resolveMarkFn() func(int) (string, error) {
	if h != nil && h.ResolveMark != nil {
		return h.ResolveMark
	}
	return resolveMarkContent
}

func resolveMarkContent(pid int) (string, error) {
	rec, err := libmark.Resolve(pid)
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", nil
	}
	return rec.Content, nil
}

func (h *Handler) spaceForWindowFn() func(windowID uint64) (int, error) {
	if h != nil && h.SpaceForWindow != nil {
		return h.SpaceForWindow
	}
	return func(windowID uint64) (int, error) {
		return spacelib.SpaceIndexForWindow(windowID)
	}
}

func (h *Handler) focusFn() func(ref shelliterm.SessionRef) error {
	if h != nil && h.Focus != nil {
		return h.Focus
	}
	return func(ref shelliterm.SessionRef) error {
		return shelliterm.Focus(ref, nil)
	}
}

func (h *Handler) switchFn() func(desktop int) error {
	if h != nil && h.Switch != nil {
		return h.Switch
	}
	return func(desktop int) error {
		return spacelib.Switch(desktop, nil)
	}
}

func (h *Handler) noteStore() *NoteStore {
	if h != nil && h.Notes != nil {
		return h.Notes
	}
	defaultNotesOnce.Do(func() {
		path, err := DefaultNotesPath()
		if err != nil {
			defaultNotes = NewNoteStore(os.TempDir() + "/iterm-bookmarks.json")
			return
		}
		defaultNotes = NewNoteStore(path)
	})
	return defaultNotes
}

func (h *Handler) spaceLabelStore() *SpaceLabelStore {
	if h != nil && h.SpaceLabels != nil {
		return h.SpaceLabels
	}
	defaultSpaceLabelsOnce.Do(func() {
		path, err := DefaultSpaceLabelsPath()
		if err != nil {
			defaultSpaceLabels = NewSpaceLabelStore(os.TempDir() + "/space-labels.json")
			return
		}
		defaultSpaceLabels = NewSpaceLabelStore(path)
	})
	return defaultSpaceLabels
}

func (h *Handler) listSpacesFn() func() ([]SpaceRef, error) {
	if h != nil && h.ListSpaces != nil {
		return h.ListSpaces
	}
	return func() ([]SpaceRef, error) {
		users, err := spacelib.ListUserSpaces()
		if err != nil {
			return nil, err
		}
		out := make([]SpaceRef, 0, len(users))
		for _, u := range users {
			out = append(out, SpaceRef{
				SpaceIndex: u.Index,
				SpaceID:    u.ID,
				UUID:       u.UUID,
				Current:    u.Current,
			})
		}
		return out, nil
	}
}

func (h *Handler) decorateInventory(inv Inventory) Inventory {
	spaces, err := h.listSpacesFn()()
	if err != nil {
		return ApplySpaceLabels(inv, nil, h.spaceLabelStore().Document())
	}
	inv = ClipDesktops(inv, spaces)
	doc := h.spaceLabelStore().Reconcile(spaces)
	return ApplySpaceLabels(inv, spaces, doc)
}

// decoratePersistHeadings clips + joins, then rewrites last-good when headings shrank/grew.
func (h *Handler) decoratePersistHeadings(raw Inventory) Inventory {
	out := h.decorateInventory(raw)
	if desktopHeadingsEqual(raw, out) {
		return out
	}
	persist := out
	persist.FromCache = false
	persist.Refreshing = false
	h.cache.mu.Lock()
	cp := persist
	h.cache.inv = &cp
	h.cache.mu.Unlock()
	h.persistInventoryCacheFile(persist)
	return out
}

// ServeHTTP handles POST /api/local/iterm2/open.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req openRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dir := strings.TrimSpace(req.Dir)
	if dir == "" {
		writeJSONError(w, http.StatusBadRequest, "dir is required")
		return
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("dir does not exist: %s", dir))
			return
		}
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("dir: %v", err))
		return
	}
	if !info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("not a directory: %s", dir))
		return
	}

	mode, err := ParseOpenMode(req.Mode)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := &shelliterm.Config{
		Mode:             mode,
		FollowUpCommands: req.Send,
		SafeInputIgnore:  true,
	}
	if err := h.openFunc()(dir, cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
