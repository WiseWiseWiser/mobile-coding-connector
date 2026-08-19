# Local iTerm2 Switcher API Doctests

Server-side inventory / focus / notes for the local macOS iTerm switcher.
Inject Capture / Switch / Focus / Notes — no live iTerm, no Mission Control.

# DSN (Domain Specific Notion)

**Participants**

- **BuildInventory** — pure join of snapshot + Desktop list + space indexes + notes.
- **FindLiveSession** — locate a session UUID in a snapshot.
- **NoteStore** — `~/.ai-critic/iterm-bookmarks.json` (tests use temp path).
- **Inventory cache file** — `~/.ai-critic/iterm-inventory-cache.json` durable last-good (tests inject `Handler.CachePath` under temp).
- **HTTP handlers** — GET inventory, POST focus, PUT notes on `server/localiterm2`.
- **Host mux** — `localiterm2.Register` mounts all three plus existing open.

**Behaviors**

- Inventory groups sessions by 0-based space index; empty Desktops still appear.
- Notes join onto live rows: agent pair (`agent_runner` + runner id) first, then `iterm_session_id`; unmatched items → `saved_notes`.
- v1 `{version:1, notes:{uuid:record}}` migrates on load to items with `iterm_session_id` only.
- iTerm not running → `iterm_running: false`, empty sessions, Desktops listed.
- Inventory never calls `space.List` / Mission Control; silent CountDesktops + SpaceIndexForWindow.
- Cold (no daemon memory **and** no readable last-good file): GET waits for a full deep capture. Stream may seed desktop headings (`snap=nil`, 0 live sessions) then full capture.
- Warm (memory exists): GET without refresh returns last-good `from_cache` and does not start a capture or layout probe. Stream must not emit an empty seed wipe; first inventory frame is last-good (`from_cache`, sessions present), then always an incremental layout-diff (not TTL skip, not full recapture of known session IDs).
- **File last-good** (disk): complete inventory is also written to `~/.ai-critic/iterm-inventory-cache.json` (tests inject `Handler.CachePath` under `t.TempDir()`). A new Handler with empty RAM loads the file so stream/GET is warm without a prior in-process GET. Missing or corrupt file is cold. Only complete last-good is persisted (not seed, not prefix-of-windows). iTerm down keeps the file (do not overwrite with empty `iterm_running=false`).
- Incremental: same session ID keeps cwd / idle / agent / notes; new ID is deep-captured alone; gone ID drops only on the final frame.
- Merge publish: no intermediate stream frame has fewer live sessions than last-good (prefix-of-windows publish is forbidden).
- `?refresh=1` still forces a full deep recapture even when memory exists.
- Focus: iTerm AppleScript select by session id only (no Desktop / Mission Control switch).
- Unknown id → 404. Switch is not part of focus.
- PUT note upserts; empty note deletes unless bookmarked; omitted fields stay put.
- PUT star persists `agent_runner` + `grok_session_id` + `iterm_session_id` when the live snap has a grok agent; otherwise `iterm_session_id` only.
- Live row JSON copies snap Agent: `Kind` → `agent_runner`; `SessionID` → `grok_session_id` when kind is grok.
- Default capture uses NoEnrich unless the notes document has a complete agent pair.
- Missing file is empty doc. Bookmarked-only records persist; gone bookmarks → saved_notes.

## Version

0.0.2

## Decision Tree

```
[local iterm2 switcher]
 |
 +-- inventory/                          (GROUP)
 |    +-- join/                          (GROUP)  pure BuildInventory
 |    |    +-- note-on-desktop-2/        (LEAF)
 |    |    +-- bookmarked-live/          (LEAF)
 |    |    +-- iterm-not-running/        (LEAF)
 |    |    +-- orphan-saved-notes/       (LEAF)
 |    |    +-- orphan-bookmarked/        (LEAF)
 |    |    +-- rematch/                  (GROUP)  v2 items + agent
 |    |         +-- agent-new-iterm/     (LEAF)
 |    |         +-- fallback-uuid/       (LEAF)
 |    |         +-- neither-orphan/      (LEAF)
 |    |         +-- iterm-id-only/       (LEAF)
 |    |         +-- grok-ids-no-cross/   (LEAF)
 |    |         +-- agent-beats-uuid/    (LEAF)
 |    |         +-- agent-incomplete-uuid/ (LEAF)
 |    |    +-- live-agent/               (GROUP)  P2 live JSON fields
 |    |         +-- copies-grok/         (LEAF)
 |    |         +-- no-agent/            (LEAF)
 |    |         +-- copies-kind-not-grok/ (LEAF)
 |    +-- handler/                       (GROUP)  GET /inventory
 |    |    +-- success/                  (LEAF)
 |    |    +-- iterm-down/               (LEAF)
 |    |    +-- no-mission-control-list/  (LEAF)
 |    +-- cache/                         (GROUP)
 |         +-- second-get-hits/          (LEAF)  GET no refresh (RAM)
 |         +-- refresh-adds-session/     (LEAF)  GET ?refresh=1
 |         +-- coalesce-refresh/         (LEAF)
 |         +-- notes-patch-bookmark/     (LEAF)
 |         +-- stream/                   (GROUP)  GET /inventory/stream
 |         |    +-- seed-first/          (LEAF)  cold seed then full
 |         |    +-- warm/                (GROUP)  daemon memory exists
 |         |         +-- last-good/      (LEAF)  first frame last-good
 |         |         +-- same-layout-no-deep/ (LEAF)
 |         |         +-- layout-adds/    (LEAF)
 |         |         +-- layout-removes/ (LEAF)
 |         |         +-- merge-no-shrink/ (LEAF)
 |         +-- file/                     (GROUP)  disk last-good CachePath
 |              +-- new-handler-loads/   (LEAF)  seed file, empty RAM
 |              +-- missing-is-cold/     (LEAF)  no file → cold seed
 |              +-- corrupt-is-cold/     (LEAF)  bad JSON → cold seed
 |              +-- write-after-complete/ (LEAF) cold GET writes file
 |              +-- iterm-down-keeps/    (LEAF)  keep file, no empty overwrite
 |    +-- enrich/                        (GROUP)  P2 conditional enrich
 |         +-- capture-uses-needs-agent/ (LEAF)
 |
 +-- focus/                              (GROUP)  POST /focus
 |    +-- known-session/                 (LEAF)
 |    +-- unknown-session/               (LEAF)
 |    +-- switch-fails/                  (LEAF)
 |
 +-- notes/                              (GROUP)
 |    +-- put-upsert/                    (LEAF)
 |    +-- put-empty-deletes/             (LEAF)
 |    +-- put-bookmark-no-note/          (LEAF)
 |    +-- put-unstar-keeps-note/         (LEAF)
 |    +-- missing-file/                  (LEAF)
 |    +-- migrate-v1-to-items/           (LEAF)
 |    +-- put-star-grok-persists-agent/  (LEAF)
 |    +-- put-star-no-agent-iterm-only/  (LEAF)
 |
 +-- register/                           (GROUP)
      +-- routes-mounted/                (LEAF)
      +-- not-in-auth-skip-list/         (LEAF)
```

## How to Run

```sh
doctest vet ./tests/local-iterm2-switcher
doctest test ./tests/local-iterm2-switcher/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	spacelib "github.com/xhd2015/dot-pkgs/go-pkgs/computer-use/macos/space"
	shelliterm "github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
	"github.com/xhd2015/ai-critic/server/localiterm2"
	"github.com/xhd2015/doctest/session"
	kooliterm "github.com/xhd2015/kool/tools/iterm2"
)

type Request struct {
	Op string // join | inventory | stream | coalesce | focus | notes | notes_store | register | skip_list | no_list_source | enrich_source

	ITermRunning bool
	CaptureErr   string
	SwitchErr    string
	FocusErr     string

	SessionID     string
	Note          string
	OmitNote      bool
	SetBookmarked bool
	Bookmarked    bool

	UseFixtureSnap bool
	WindowSpace    int // space index for fixture window 42
	DesktopN       int // CountDesktops inject; 0 → 2
	NotesJSON      string
	NotesPath      string
	// Agent on fixture sess-a (injected; no live procresolve).
	AgentKind      string
	AgentSessionID string

	// cache leaves
	DoSecondGET  bool
	DoRefreshGET bool
	DoNotesAfterGET bool
	AfterNoteBookmarked bool
	SecondSnapB  bool // later deep Capture adds sess-b (mutates sess-a cwd)
	FirstSnapAB      bool // first Capture is sess-a + sess-b
	SecondSnapAOnly  bool // later deep Capture is sess-a only
	UseTwoWindowSnap bool // sess-a and sess-b in separate windows (prefix publish)
	CoalesceN    int
	CaptureHold  chan struct{}
	CaptureEntered chan struct{}

	// file last-good (disk cache under Handler.CachePath)
	SeedCacheJSON        string // write to CachePath before constructing Handler
	NewHandlerAfterWrite bool   // GET on handler A, then measure on new Handler (same CachePath, empty RAM)
	DoGETAfterStream     bool   // after stream, also GET without refresh (fill FromCache from GET)
}

type Response struct {
	StatusCode int
	Body       string
	Error      string
	OK         bool

	ITermRunning bool
	DesktopCount int
	SessionCount int
	SavedCount   int
	NoteOnFirst  string
	FirstDesktop int
	FirstSpace   int
	HasOrphan    bool
	OrphanNote   string
	FromCache    bool
	CachedAt     string
	CaptureCalls int
	LayoutCalls  int
	HasSessionA  bool
	HasSessionB  bool
	SessionACwd  string
	SessionBCwd  string
	FirstFrameFromCache     bool
	FirstFrameSessionCount  int
	FirstFrameDesktopCount  int
	FirstFrameHasSessionA   bool
	FirstFrameHasSessionB   bool
	MinFrameSessionCount    int
	MinNonFinalSessionCount int
	ListCalled   bool
	UsesSpaceList bool
	StreamFrames int
	StreamMounted bool

	SwitchCalled  bool
	SwitchDesktop int
	FocusCalled   bool
	FocusSession  string
	FocusTab      int
	FocusWindow   string

	StoredNote           string
	StoredBookmarked     bool
	HasRecord            bool
	HasFile              bool
	FirstBookmarked      bool
	BookmarkCount        int
	DocVersion           int
	StoredItemsCount     int
	StoredAgentRunner    string
	StoredGrokSessionID  string
	StoredITermSessionID string

	InventoryMounted bool
	FocusMounted     bool
	NotesMounted     bool
	InAuthSkipList   bool
	RegisteredInServer bool
	SkipListSource   string

	FirstAgentRunner    string
	FirstGrokSessionID  string
	HasNeedsAgentEnrich bool
	HardcodedNoEnrichOnly bool

	// disk last-good probe (Handler.CachePath under t.TempDir)
	CacheFileExists       bool
	CacheFileHasSessionA  bool
	CacheFileHasSessionB  bool
	CacheFileSessionCount int
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "join":
		return runJoin(t, req, resp)
	case "inventory":
		return runInventory(t, req, resp)
	case "stream":
		return runStream(t, req, resp)
	case "no_list_source":
		return runNoListSource(t, d, resp)
	case "enrich_source":
		return runEnrichSource(t, d, resp)
	case "focus":
		return runFocus(t, req, resp)
	case "notes", "notes_store":
		return runNotes(t, req, resp)
	case "register":
		return runRegister(t, req, resp)
	case "skip_list":
		return runSkipList(t, d, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func fixtureSnap(req *Request) *kooliterm.Snapshot {
	cwd := "/Users/xhd2015/proj/ai-critic"
	idle := false
	sess := kooliterm.SnapshotSession{
		Index: 1,
		ID:    "sess-a",
		Name:  "grok review",
		Cwd:   &cwd,
		Idle:  &idle,
	}
	if req != nil && (req.AgentKind != "" || req.AgentSessionID != "") {
		sess.Agent = &kooliterm.SessionAgent{
			Kind:      req.AgentKind,
			SessionID: req.AgentSessionID,
		}
	}
	return &kooliterm.Snapshot{
		Windows: []kooliterm.SnapshotWindow{{
			Index:    1,
			Name:     "ai-critic",
			WindowID: 42,
			Tabs: []kooliterm.SnapshotTab{{
				Index:    2,
				Name:     "grok",
				Sessions: []kooliterm.SnapshotSession{sess},
			}},
		}},
	}
}

func fixtureSnapB(req *Request) *kooliterm.Snapshot {
	snap := fixtureSnap(req)
	cwd := "/tmp/other"
	idle := true
	snap.Windows[0].Tabs[0].Sessions = append(snap.Windows[0].Tabs[0].Sessions, kooliterm.SnapshotSession{
		Index: 2,
		ID:    "sess-b",
		Name:  "new tab",
		Cwd:   &cwd,
		Idle:  &idle,
	})
	return snap
}

func fixtureSnapBMutatedA(req *Request) *kooliterm.Snapshot {
	snap := fixtureSnapB(req)
	cwd := "/tmp/recaptured-a"
	snap.Windows[0].Tabs[0].Sessions[0].Cwd = &cwd
	return snap
}

func fixtureTwoWindows(req *Request) *kooliterm.Snapshot {
	snap := fixtureSnap(req)
	cwd := "/tmp/other"
	idle := true
	snap.Windows = append(snap.Windows, kooliterm.SnapshotWindow{
		Index:    2,
		Name:     "other",
		WindowID: 43,
		Tabs: []kooliterm.SnapshotTab{{
			Index: 1,
			Name:  "sh",
			Sessions: []kooliterm.SnapshotSession{{
				Index: 1,
				ID:    "sess-b",
				Name:  "other pane",
				Cwd:   &cwd,
				Idle:  &idle,
			}},
		}},
	})
	return snap
}

func layoutIDsOnly(snap *kooliterm.Snapshot) *kooliterm.Snapshot {
	if snap == nil {
		return nil
	}
	out := *snap
	out.Windows = make([]kooliterm.SnapshotWindow, len(snap.Windows))
	for i, w := range snap.Windows {
		nw := w
		nw.Tabs = make([]kooliterm.SnapshotTab, len(w.Tabs))
		for j, tab := range w.Tabs {
			nt := tab
			nt.Sessions = make([]kooliterm.SnapshotSession, len(tab.Sessions))
			for k, s := range tab.Sessions {
				nt.Sessions[k] = kooliterm.SnapshotSession{
					Index: s.Index,
					ID:    s.ID,
					Name:  s.Name,
				}
			}
			nw.Tabs[j] = nt
		}
		out.Windows[i] = nw
	}
	return &out
}

func inventorySessionCount(inv localiterm2.Inventory) int {
	n := 0
	for _, d := range inv.Desktops {
		n += len(d.Sessions)
	}
	return n
}

func inventoryHasSession(inv localiterm2.Inventory, id string) bool {
	for _, d := range inv.Desktops {
		for _, s := range d.Sessions {
			if s.SessionID == id {
				return true
			}
		}
	}
	return false
}

func fixtureDesktops() []spacelib.Desktop {
	return []spacelib.Desktop{
		{Number: 1, Name: "Desktop 1"},
		{Number: 2, Name: "Desktop 2"},
	}
}

func runJoin(t *testing.T, req *Request, resp *Response) (*Response, error) {
	notes := localiterm2.NotesDocument{Version: 1, Notes: map[string]*localiterm2.NoteRecord{}}
	if req.NotesJSON != "" {
		if err := json.Unmarshal([]byte(req.NotesJSON), &notes); err != nil {
			return nil, err
		}
	}
	var snap *kooliterm.Snapshot
	spaces := map[uint64]int{}
	if req.ITermRunning {
		snap = fixtureSnap(req)
		spaces[42] = req.WindowSpace
	}
	inv := localiterm2.BuildInventory(snap, fixtureDesktops(), spaces, &notes, req.ITermRunning)
	fillInventory(resp, inv)
	return resp, nil
}

func serveOn(h *localiterm2.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	localiterm2.Register(mux, h)
	var rdr *bytes.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	} else {
		rdr = bytes.NewReader(nil)
	}
	httpReq := httptest.NewRequest(method, path, rdr)
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httpReq)
	return rr
}

func runInventory(t *testing.T, req *Request, resp *Response) (*Response, error) {
	h, cap := testHandler(t, req)
	if req.NewHandlerAfterWrite {
		// Handler A completes a GET (writes complete last-good to CachePath when implemented).
		_ = serveOn(h, http.MethodGet, localiterm2.InventoryPath, nil)
		// Handler B: empty RAM, same CachePath (no re-seed; file is the only warm source).
		h, cap = testHandlerAt(t, req, cap.cachePath, false)
	}
	rr := serveOn(h, http.MethodGet, localiterm2.InventoryPath, nil)
	fillHTTP(resp, rr)
	var inv localiterm2.Inventory
	if json.Unmarshal(rr.Body.Bytes(), &inv) == nil {
		fillInventory(resp, inv)
	}
	if req.DoSecondGET {
		rr2 := serveOn(h, http.MethodGet, localiterm2.InventoryPath, nil)
		fillHTTP(resp, rr2)
		if json.Unmarshal(rr2.Body.Bytes(), &inv) == nil {
			fillInventory(resp, inv)
		}
	}
	if req.DoNotesAfterGET {
		payload := map[string]any{"session_id": req.SessionID, "bookmarked": req.AfterNoteBookmarked}
		body, _ := json.Marshal(payload)
		rrNote := serveOn(h, http.MethodPut, localiterm2.NotesPath, body)
		fillHTTP(resp, rrNote)
		rr2 := serveOn(h, http.MethodGet, localiterm2.InventoryPath, nil)
		if json.Unmarshal(rr2.Body.Bytes(), &inv) == nil {
			fillInventory(resp, inv)
		}
	}
	if req.DoRefreshGET {
		rr3 := serveOn(h, http.MethodGet, localiterm2.InventoryPath+"?refresh=1", nil)
		fillHTTP(resp, rr3)
		if json.Unmarshal(rr3.Body.Bytes(), &inv) == nil {
			fillInventory(resp, inv)
		}
	}
	if req.CoalesceN > 0 {
		var wg sync.WaitGroup
		started := make(chan struct{}, req.CoalesceN)
		for i := 0; i < req.CoalesceN; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				started <- struct{}{}
				serveOn(h, http.MethodGet, localiterm2.InventoryPath+"?refresh=1", nil)
			}()
		}
		for i := 0; i < req.CoalesceN; i++ {
			<-started
		}
		if req.CaptureEntered != nil {
			<-req.CaptureEntered
		}
		// Let the other refresh callers join waitRefresh before we release Capture.
		time.Sleep(30 * time.Millisecond)
		if req.CaptureHold != nil {
			close(req.CaptureHold)
		}
		wg.Wait()
	}
	resp.CaptureCalls = cap.captureCalls
	resp.LayoutCalls = cap.layoutCalls
	resp.ListCalled = cap.listCalled
	fillCacheFile(resp, cap.cachePath)
	return resp, nil
}

func runStream(t *testing.T, req *Request, resp *Response) (*Response, error) {
	h, cap := testHandler(t, req)
	if req.NewHandlerAfterWrite {
		_ = serveOn(h, http.MethodGet, localiterm2.InventoryPath, nil)
		h, cap = testHandlerAt(t, req, cap.cachePath, false)
	}
	if req.DoSecondGET {
		// Warm RAM cache first so the stream can emit a from_cache frame.
		_ = serveOn(h, http.MethodGet, localiterm2.InventoryPath, nil)
	}
	rr := serveOn(h, http.MethodGet, localiterm2.InventoryStreamPath, nil)
	fillHTTP(resp, rr)
	frames := parseInventorySSE(rr.Body.String())
	resp.StreamFrames = len(frames)
	if len(frames) > 0 {
		resp.FirstFrameFromCache = frames[0].FromCache
		resp.FirstFrameSessionCount = inventorySessionCount(frames[0])
		resp.FirstFrameDesktopCount = len(frames[0].Desktops)
		resp.FirstFrameHasSessionA = inventoryHasSession(frames[0], "sess-a")
		resp.FirstFrameHasSessionB = inventoryHasSession(frames[0], "sess-b")
		minAll := inventorySessionCount(frames[0])
		minNonFinal := minAll
		for i, fr := range frames {
			n := inventorySessionCount(fr)
			if n < minAll {
				minAll = n
			}
			if i < len(frames)-1 && n < minNonFinal {
				minNonFinal = n
			}
		}
		resp.MinFrameSessionCount = minAll
		if len(frames) == 1 {
			resp.MinNonFinalSessionCount = minAll
		} else {
			resp.MinNonFinalSessionCount = minNonFinal
		}
		fillInventory(resp, frames[len(frames)-1])
	}
	if req.DoGETAfterStream {
		// Measure warm GET on the same Handler (file load should make CaptureCalls stay 0).
		rr2 := serveOn(h, http.MethodGet, localiterm2.InventoryPath, nil)
		fillHTTP(resp, rr2)
		var inv localiterm2.Inventory
		if json.Unmarshal(rr2.Body.Bytes(), &inv) == nil {
			// Keep FirstFrame* from stream; overwrite GET-facing inventory fields.
			fillInventory(resp, inv)
		}
	}
	resp.CaptureCalls = cap.captureCalls
	resp.LayoutCalls = cap.layoutCalls
	fillCacheFile(resp, cap.cachePath)
	return resp, nil
}

func parseInventorySSE(body string) []localiterm2.Inventory {
	var out []localiterm2.Inventory
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		raw := strings.TrimPrefix(line, "data: ")
		var wrap struct {
			Type      string                 `json:"type"`
			Inventory *localiterm2.Inventory `json:"inventory"`
		}
		if json.Unmarshal([]byte(raw), &wrap) != nil || wrap.Type != "inventory" || wrap.Inventory == nil {
			continue
		}
		out = append(out, *wrap.Inventory)
	}
	return out
}

func runNoListSource(t *testing.T, d *session.Doctest, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot(d)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(moduleRoot, "server", "localiterm2")
	combined := ""
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		combined += string(b)
	}
	resp.UsesSpaceList = strings.Contains(combined, "spacelib.List(") ||
		strings.Contains(combined, "space.List(") ||
		strings.Contains(combined, "ListDesktops")
	return resp, nil
}

func runEnrichSource(t *testing.T, d *session.Doctest, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot(d)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(moduleRoot, "server", "localiterm2")
	combined := ""
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		combined += string(b)
	}
	hasHelper := strings.Contains(combined, "NeedsAgentEnrich") ||
		strings.Contains(combined, "needsAgentEnrich")
	resp.HasNeedsAgentEnrich = hasHelper
	resp.HardcodedNoEnrichOnly = strings.Contains(combined, "NoEnrich: true") && !hasHelper
	return resp, nil
}

func runFocus(t *testing.T, req *Request, resp *Response) (*Response, error) {
	h, cap := testHandler(t, req)
	body, _ := json.Marshal(map[string]string{"session_id": req.SessionID})
	rr := serveOn(h, http.MethodPost, localiterm2.FocusPath, body)
	fillHTTP(resp, rr)
	resp.SwitchCalled = cap.switchCalled
	resp.SwitchDesktop = cap.switchDesktop
	resp.FocusCalled = cap.focusCalled
	resp.FocusSession = cap.focusSession
	resp.FocusTab = cap.focusTab
	resp.FocusWindow = cap.focusWindow
	return resp, nil
}

func runNotes(t *testing.T, req *Request, resp *Response) (*Response, error) {
	h, _ := testHandler(t, req)
	if req.Op == "notes_store" {
		doc := h.Notes.Document()
		fillStoredFromDoc(resp, doc, req.SessionID)
		_, err := os.Stat(h.Notes.Path())
		resp.HasFile = err == nil
		if req.ITermRunning {
			spaces := map[uint64]int{42: req.WindowSpace}
			inv := localiterm2.BuildInventory(fixtureSnap(req), fixtureDesktops(), spaces, doc, true)
			fillInventory(resp, inv)
		}
		return resp, nil
	}
	payload := map[string]any{"session_id": req.SessionID}
	if !req.OmitNote {
		payload["note"] = req.Note
	}
	if req.SetBookmarked {
		payload["bookmarked"] = req.Bookmarked
	}
	body, _ := json.Marshal(payload)
	rr := serveOn(h, http.MethodPut, localiterm2.NotesPath, body)
	fillHTTP(resp, rr)
	if rec := h.Notes.Get(req.SessionID); rec != nil {
		resp.StoredNote = rec.Note
		resp.StoredBookmarked = rec.Bookmarked
		resp.HasRecord = true
	}
	fillStoredFromDoc(resp, h.Notes.Document(), req.SessionID)
	_, err := os.Stat(h.Notes.Path())
	resp.HasFile = err == nil
	return resp, nil
}

type storedItemProbe struct {
	Note           string `json:"note"`
	Bookmarked     bool   `json:"bookmarked"`
	AgentRunner    string `json:"agent_runner"`
	GrokSessionID  string `json:"grok_session_id"`
	ITermSessionID string `json:"iterm_session_id"`
}

func fillStoredFromDoc(resp *Response, doc *localiterm2.NotesDocument, sessionID string) {
	if doc == nil {
		return
	}
	resp.DocVersion = doc.Version
	if rec := doc.Notes[sessionID]; rec != nil {
		resp.StoredNote = rec.Note
		resp.StoredBookmarked = rec.Bookmarked
		resp.HasRecord = true
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return
	}
	var probe struct {
		Version int               `json:"version"`
		Items   []storedItemProbe `json:"items"`
	}
	if json.Unmarshal(raw, &probe) != nil {
		return
	}
	resp.DocVersion = probe.Version
	resp.StoredItemsCount = len(probe.Items)
	item := pickStoredItem(probe.Items, sessionID)
	if item == nil {
		return
	}
	resp.StoredAgentRunner = item.AgentRunner
	resp.StoredGrokSessionID = item.GrokSessionID
	resp.StoredITermSessionID = item.ITermSessionID
	if !resp.HasRecord {
		resp.StoredNote = item.Note
		resp.StoredBookmarked = item.Bookmarked
		resp.HasRecord = true
	}
}

func pickStoredItem(items []storedItemProbe, sessionID string) *storedItemProbe {
	for i := range items {
		if items[i].ITermSessionID == sessionID {
			return &items[i]
		}
	}
	if sessionID == "" && len(items) == 1 {
		return &items[0]
	}
	if len(items) == 1 {
		return &items[0]
	}
	return nil
}

func runRegister(t *testing.T, req *Request, resp *Response) (*Response, error) {
	h, _ := testHandler(t, req)
	mux := http.NewServeMux()
	localiterm2.Register(mux, h)

	invReq := httptest.NewRequest(http.MethodGet, localiterm2.InventoryPath, nil)
	invRR := httptest.NewRecorder()
	mux.ServeHTTP(invRR, invReq)
	resp.InventoryMounted = invRR.Code != 404

	focusBody, _ := json.Marshal(map[string]string{"session_id": "sess-a"})
	focusReq := httptest.NewRequest(http.MethodPost, localiterm2.FocusPath, bytes.NewReader(focusBody))
	focusRR := httptest.NewRecorder()
	mux.ServeHTTP(focusRR, focusReq)
	resp.FocusMounted = focusRR.Code != 404

	notesBody, _ := json.Marshal(map[string]string{"session_id": "sess-a", "note": "x"})
	notesReq := httptest.NewRequest(http.MethodPut, localiterm2.NotesPath, bytes.NewReader(notesBody))
	notesRR := httptest.NewRecorder()
	mux.ServeHTTP(notesRR, notesReq)
	resp.NotesMounted = notesRR.Code != 404

	streamReq := httptest.NewRequest(http.MethodGet, localiterm2.InventoryStreamPath, nil)
	streamRR := httptest.NewRecorder()
	mux.ServeHTTP(streamRR, streamReq)
	resp.StreamMounted = streamRR.Code != 404

	resp.StatusCode = invRR.Code
	return resp, nil
}

func runSkipList(t *testing.T, d *session.Doctest, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot(d)
	if err != nil {
		return nil, err
	}
	srcPath := filepath.Join(moduleRoot, "server", "server.go")
	b, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}
	src := string(b)
	resp.SkipListSource = srcPath
	resp.RegisteredInServer = strings.Contains(src, "localiterm2.Register")
	idx := strings.Index(src, "auth.Middleware")
	window := src
	if idx >= 0 {
		window = src[idx:]
		if len(window) > 3000 {
			window = window[:3000]
		}
	}
	resp.InAuthSkipList = strings.Contains(window, `"`+localiterm2.InventoryPath+`"`) ||
		strings.Contains(window, `"`+localiterm2.FocusPath+`"`) ||
		strings.Contains(window, `"`+localiterm2.NotesPath+`"`)
	return resp, nil
}

type hookCap struct {
	switchCalled  bool
	switchDesktop int
	focusCalled   bool
	focusSession  string
	focusTab      int
	focusWindow   string
	captureCalls  int
	layoutCalls   int
	listCalled    bool
	cachePath     string // injectable inventory last-good file (never $HOME)
}

// fixtureLastGoodCacheJSON is a complete last-good Inventory for disk-cache seeds.
// Fixture IDs match existing leaves: sess-a, cwd /Users/xhd2015/proj/ai-critic.
func fixtureLastGoodCacheJSON() string {
	return `{
  "iterm_running": true,
  "cached_at": "2026-08-14T12:00:00Z",
  "from_cache": false,
  "refreshing": false,
  "desktops": [
    {
      "space_index": 0,
      "desktop": 1,
      "sessions": [
        {
          "session_id": "sess-a",
          "session_name": "grok review",
          "window_id": "42",
          "window_name": "ai-critic",
          "tab_index": 2,
          "tab_name": "grok",
          "cwd": "/Users/xhd2015/proj/ai-critic",
          "idle": false,
          "bookmarked": false,
          "space_index": 0,
          "desktop": 1
        }
      ]
    },
    {
      "space_index": 1,
      "desktop": 2,
      "sessions": []
    }
  ],
  "saved_notes": []
}`
}

// testHandler builds an injected Handler. Free function: doctest drops methods.
// Always sets CachePath under t.TempDir so future disk persist cannot touch $HOME.
func testHandler(t *testing.T, req *Request) (*localiterm2.Handler, *hookCap) {
	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, "iterm-inventory-cache.json")
	return testHandlerAt(t, req, cachePath, true)
}

// testHandlerAt builds a Handler with a fixed CachePath.
// writeSeed controls whether SeedCacheJSON is written (false for NewHandlerAfterWrite B).
func testHandlerAt(t *testing.T, req *Request, cachePath string, writeSeed bool) (*localiterm2.Handler, *hookCap) {
	cap := &hookCap{cachePath: cachePath}
	tmp := filepath.Dir(cachePath)
	if tmp == "" || tmp == "." {
		tmp = t.TempDir()
		cachePath = filepath.Join(tmp, "iterm-inventory-cache.json")
		cap.cachePath = cachePath
	}
	store := localiterm2.NewNoteStore(filepath.Join(tmp, "iterm-bookmarks.json"))
	if req.NotesPath != "" {
		store = localiterm2.NewNoteStore(req.NotesPath)
	}
	if req.NotesJSON != "" {
		path := store.Path()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(req.NotesJSON), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if writeSeed && req.SeedCacheJSON != "" {
		if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cachePath, []byte(req.SeedCacheJSON), 0644); err != nil {
			t.Fatal(err)
		}
	}
	h := &localiterm2.Handler{
		Notes: store,
		Now:   func() time.Time { return time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC) },
		ITermRunning: func() bool {
			return req.ITermRunning
		},
		Capture: func() (*kooliterm.Snapshot, error) {
			cap.captureCalls++
			if req.CaptureHold != nil && cap.captureCalls >= 2 {
				if req.CaptureEntered != nil {
					select {
					case req.CaptureEntered <- struct{}{}:
					default:
					}
				}
				<-req.CaptureHold
			}
			if req.CaptureErr != "" {
				return nil, fmt.Errorf("%s", req.CaptureErr)
			}
			if !req.ITermRunning && req.CaptureErr == "" && !req.UseFixtureSnap {
				return nil, fmt.Errorf("Error: iTerm2 is not running")
			}
			if req.SecondSnapB && cap.captureCalls >= 2 {
				return fixtureSnapBMutatedA(req), nil
			}
			if req.SecondSnapAOnly && cap.captureCalls >= 2 {
				return fixtureSnap(req), nil
			}
			if req.UseTwoWindowSnap {
				return fixtureTwoWindows(req), nil
			}
			if req.FirstSnapAB {
				return fixtureSnapB(req), nil
			}
			return fixtureSnap(req), nil
		},
		CountDesktops: func() (int, error) {
			if req.DesktopN > 0 {
				return req.DesktopN, nil
			}
			return 2, nil
		},
		ListSpaces: func() ([]localiterm2.SpaceRef, error) {
			n := 2
			if req.DesktopN > 0 {
				n = req.DesktopN
			}
			out := make([]localiterm2.SpaceRef, n)
			for i := 0; i < n; i++ {
				out[i] = localiterm2.SpaceRef{SpaceIndex: i, SpaceID: uint64(100 + i)}
			}
			return out, nil
		},
		SpaceLabels: localiterm2.NewSpaceLabelStore(filepath.Join(tmp, "space-labels.json")),
		SpaceForWindow: func(windowID uint64) (int, error) {
			return req.WindowSpace, nil
		},
		Switch: func(desktop int) error {
			cap.switchCalled = true
			cap.switchDesktop = desktop
			if req.SwitchErr != "" {
				return fmt.Errorf("%s", req.SwitchErr)
			}
			return nil
		},
		Focus: func(ref shelliterm.SessionRef) error {
			cap.focusCalled = true
			cap.focusSession = ref.SessionID
			cap.focusTab = ref.TabIndex
			cap.focusWindow = ref.WindowID
			if req.FocusErr != "" {
				return fmt.Errorf("%s", req.FocusErr)
			}
			return nil
		},
	}
	// CachePath is injectable disk last-good path (implementer adds the field).
	// Wired via reflection so this tree compiles before CachePath exists.
	setHandlerString(h, "CachePath", cachePath)
	// Layout / ProbeLayout is the incremental layout-diff hook the implementer adds.
	// Wired via reflection so this tree compiles before the field exists.
	setHandlerLayout(h, func() (*kooliterm.Snapshot, error) {
		cap.layoutCalls++
		var snap *kooliterm.Snapshot
		switch {
		case req.SecondSnapB:
			snap = fixtureSnapB(req)
		case req.SecondSnapAOnly:
			snap = fixtureSnap(req)
		case req.UseTwoWindowSnap:
			snap = fixtureTwoWindows(req)
		case req.FirstSnapAB:
			snap = fixtureSnapB(req)
		default:
			snap = fixtureSnap(req)
		}
		return layoutIDsOnly(snap), nil
	})
	return h, cap
}

func setHandlerString(h *localiterm2.Handler, field, value string) {
	if h == nil {
		return
	}
	rv := reflect.ValueOf(h).Elem()
	f := rv.FieldByName(field)
	if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.String {
		return
	}
	f.SetString(value)
}

func setHandlerLayout(h *localiterm2.Handler, fn func() (*kooliterm.Snapshot, error)) {
	if h == nil || fn == nil {
		return
	}
	rv := reflect.ValueOf(h).Elem()
	fv := reflect.ValueOf(fn)
	for _, name := range []string{"Layout", "ProbeLayout", "LayoutProbe", "CaptureLayout"} {
		f := rv.FieldByName(name)
		if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.Func {
			continue
		}
		if fv.Type().AssignableTo(f.Type()) {
			f.Set(fv)
			return
		}
	}
}

func fillCacheFile(resp *Response, path string) {
	if path == "" {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		resp.CacheFileExists = false
		return
	}
	resp.CacheFileExists = true
	var inv localiterm2.Inventory
	if json.Unmarshal(b, &inv) != nil {
		return
	}
	resp.CacheFileSessionCount = inventorySessionCount(inv)
	resp.CacheFileHasSessionA = inventoryHasSession(inv, "sess-a")
	resp.CacheFileHasSessionB = inventoryHasSession(inv, "sess-b")
}

func fillInventory(resp *Response, inv localiterm2.Inventory) {
	resp.ITermRunning = inv.ITermRunning
	resp.DesktopCount = len(inv.Desktops)
	resp.SavedCount = len(inv.SavedNotes)
	n := 0
	bookmarks := 0
	for _, d := range inv.Desktops {
		n += len(d.Sessions)
		for _, s := range d.Sessions {
			if s.Bookmarked {
				bookmarks++
			}
		}
	}
	resp.SessionCount = n
	resp.BookmarkCount = bookmarks
	if len(inv.Desktops) > 0 {
		resp.FirstDesktop = inv.Desktops[0].Desktop
		resp.FirstSpace = inv.Desktops[0].SpaceIndex
	}
	for _, d := range inv.Desktops {
		if len(d.Sessions) > 0 {
			resp.NoteOnFirst = d.Sessions[0].Note
			resp.FirstBookmarked = d.Sessions[0].Bookmarked
			resp.FirstDesktop = d.Desktop
			resp.FirstSpace = d.SpaceIndex
			raw, _ := json.Marshal(d.Sessions[0])
			var probe struct {
				AgentRunner   string `json:"agent_runner"`
				GrokSessionID string `json:"grok_session_id"`
			}
			_ = json.Unmarshal(raw, &probe)
			resp.FirstAgentRunner = probe.AgentRunner
			resp.FirstGrokSessionID = probe.GrokSessionID
			break
		}
	}
	if len(inv.SavedNotes) > 0 {
		resp.HasOrphan = true
		resp.OrphanNote = inv.SavedNotes[0].Note
	}
	resp.FromCache = inv.FromCache
	resp.CachedAt = inv.CachedAt
	resp.HasSessionA = false
	resp.HasSessionB = false
	resp.SessionACwd = ""
	resp.SessionBCwd = ""
	for _, d := range inv.Desktops {
		for _, s := range d.Sessions {
			if s.SessionID == "sess-a" {
				resp.HasSessionA = true
				resp.SessionACwd = s.Cwd
			}
			if s.SessionID == "sess-b" {
				resp.HasSessionB = true
				resp.SessionBCwd = s.Cwd
			}
		}
	}
}

func fillHTTP(resp *Response, rr *httptest.ResponseRecorder) {
	resp.StatusCode = rr.Code
	resp.Body = rr.Body.String()
	var m map[string]any
	if json.Unmarshal(rr.Body.Bytes(), &m) == nil {
		if e, ok := m["error"].(string); ok {
			resp.Error = e
		}
		if ok, exists := m["ok"].(bool); exists {
			resp.OK = ok
		}
	}
}

func findModuleRoot(d *session.Doctest) (string, error) {
	if d != nil && d.DOCTEST_ROOT != "" {
		for dir := d.DOCTEST_ROOT; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
```
