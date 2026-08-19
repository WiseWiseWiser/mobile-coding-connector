package localiterm2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	spacelib "github.com/xhd2015/dot-pkgs/go-pkgs/computer-use/macos/space"
	kooliterm "github.com/xhd2015/kool/tools/iterm2"
)

// inventoryCache is in-process last-good. Complete snapshots also persist to CachePath.
type inventoryCache struct {
	mu         sync.Mutex
	inv        *Inventory
	refreshing bool
	wait       chan struct{} // closed when the current single-flight refresh finishes
	lastErr    error
	emits      []func(Inventory)
}

// DefaultInventoryCachePath returns ~/.ai-critic/iterm-inventory-cache.json.
func DefaultInventoryCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", "iterm-inventory-cache.json"), nil
}

// BuildInventory joins a snapshot, Desktop list, per-window space indexes, and notes.
// When snap is nil or itermRunning is false, Desktops still appear (empty sessions).
// Empty Desktop lists are synthesized from window spaces, or Desktop 1.
// Does not read ~/.mark; Handler paths pass resolveMark via assembleInventory.
func BuildInventory(snap *kooliterm.Snapshot, desktops []spacelib.Desktop, spaceByWindow map[uint64]int, notes *NotesDocument, itermRunning bool) Inventory {
	return buildInventory(snap, desktops, spaceByWindow, notes, itermRunning, nil)
}

func (h *Handler) assembleInventory(snap *kooliterm.Snapshot, desktops []spacelib.Desktop, spaceByWindow map[uint64]int, notes *NotesDocument, itermRunning bool) Inventory {
	return buildInventory(snap, desktops, spaceByWindow, notes, itermRunning, h.resolveMarkFn())
}

func buildInventory(snap *kooliterm.Snapshot, desktops []spacelib.Desktop, spaceByWindow map[uint64]int, notes *NotesDocument, itermRunning bool, resolveMark func(int) (string, error)) Inventory {
	if notes == nil {
		notes = emptyNotes()
	}
	if spaceByWindow == nil {
		spaceByWindow = map[uint64]int{}
	}

	groups := map[int]*DesktopGroup{}
	ensure := func(spaceIdx int) *DesktopGroup {
		if spaceIdx < 0 {
			spaceIdx = 0
		}
		g, ok := groups[spaceIdx]
		if !ok {
			g = &DesktopGroup{
				SpaceIndex: spaceIdx,
				Desktop:    spaceIdx + 1,
				Sessions:   []LiveSession{},
			}
			groups[spaceIdx] = g
		}
		return g
	}

	listedMax := -1
	for _, d := range desktops {
		n := d.Number
		if n < 1 {
			continue
		}
		idx := n - 1
		ensure(idx)
		if idx > listedMax {
			listedMax = idx
		}
	}

	var lives []liveJoinKey
	if itermRunning && snap != nil {
		for _, win := range snap.Windows {
			spaceIdx := 0
			if win.WindowID != 0 {
				if idx, ok := spaceByWindow[win.WindowID]; ok && idx >= 0 {
					spaceIdx = idx
				}
			}
			if listedMax >= 0 && spaceIdx > listedMax {
				spaceIdx = listedMax
			}
			g := ensure(spaceIdx)
			for _, tab := range win.Tabs {
				for _, sess := range tab.Sessions {
					id := strings.TrimSpace(sess.ID)
					if id == "" {
						continue
					}
					ls := liveSessionFromSnap(sess, win, tab, spaceIdx, resolveMark)
					lives = append(lives, liveJoinKey{
						sessionID:      ls.SessionID,
						agentKind:      ls.agentKind,
						agentSessionID: ls.agentSessionID,
					})
					g.Sessions = append(g.Sessions, ls)
				}
			}
		}
	}
	matched, unused := matchNoteItems(notesItems(notes), lives)
	for _, g := range groups {
		for i := range g.Sessions {
			if it := matched[g.Sessions[i].SessionID]; it != nil {
				g.Sessions[i].Note = it.Note
				g.Sessions[i].Bookmarked = it.Bookmarked
			}
		}
	}

	if len(groups) == 0 {
		ensure(0)
	}

	idxs := make([]int, 0, len(groups))
	for i := range groups {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	out := Inventory{
		ITermRunning: itermRunning,
		Desktops:     make([]DesktopGroup, 0, len(idxs)),
		SavedNotes:   nil,
	}
	for _, i := range idxs {
		out.Desktops = append(out.Desktops, *groups[i])
	}

	out.SavedNotes = orphansFromItems(unused)
	return out
}

// ClipDesktops keeps headings aligned with live Mission Control Spaces.
// Empty live (list failed) leaves inv unchanged. Does not join labels.
func ClipDesktops(inv Inventory, live []SpaceRef) Inventory {
	if len(live) == 0 {
		return inv
	}
	byIndex := make(map[int]DesktopGroup, len(inv.Desktops))
	for _, g := range inv.Desktops {
		byIndex[g.SpaceIndex] = g
	}
	out := make([]DesktopGroup, 0, len(live))
	for _, ref := range live {
		if g, ok := byIndex[ref.SpaceIndex]; ok {
			out = append(out, g)
			continue
		}
		out = append(out, DesktopGroup{
			SpaceIndex: ref.SpaceIndex,
			Desktop:    ref.SpaceIndex + 1,
			Sessions:   []LiveSession{},
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].SpaceIndex < out[j].SpaceIndex
	})
	inv.Desktops = out
	return inv
}

func desktopHeadingsEqual(a, b Inventory) bool {
	if len(a.Desktops) != len(b.Desktops) {
		return false
	}
	for i := range a.Desktops {
		if a.Desktops[i].SpaceIndex != b.Desktops[i].SpaceIndex {
			return false
		}
	}
	return true
}

// ApplyNotes overlays the current notes file onto an existing inventory
// without recapturing iTerm. Used after PUT /notes so the cache is not stale.
func ApplyNotes(inv Inventory, notes *NotesDocument) Inventory {
	if notes == nil {
		notes = emptyNotes()
	}
	var lives []liveJoinKey
	desktops := make([]DesktopGroup, len(inv.Desktops))
	for i, g := range inv.Desktops {
		sessions := make([]LiveSession, len(g.Sessions))
		copy(sessions, g.Sessions)
		for j := range sessions {
			lives = append(lives, liveJoinKey{
				sessionID:      sessions[j].SessionID,
				agentKind:      sessions[j].agentKind,
				agentSessionID: sessions[j].agentSessionID,
			})
		}
		g.Sessions = sessions
		desktops[i] = g
	}
	matched, unused := matchNoteItems(notesItems(notes), lives)
	for i := range desktops {
		for j := range desktops[i].Sessions {
			s := &desktops[i].Sessions[j]
			if it := matched[s.SessionID]; it != nil {
				s.Note = it.Note
				s.Bookmarked = it.Bookmarked
			} else {
				s.Note = ""
				s.Bookmarked = false
			}
		}
	}
	inv.Desktops = desktops
	inv.SavedNotes = orphansFromItems(unused)
	return inv
}

type liveJoinKey struct {
	sessionID      string
	agentKind      string
	agentSessionID string
}

// matchNoteItems joins v2 items onto live panes: complete agent pair first
// (first hit wins), then iterm_session_id. Unused keep-worthy items are orphans.
func matchNoteItems(items []*NoteItem, lives []liveJoinKey) (map[string]*NoteItem, []*NoteItem) {
	used := make([]bool, len(items))
	bySession := make(map[string]*NoteItem, len(lives))

	for _, live := range lives {
		if live.agentKind == "" || live.agentSessionID == "" {
			continue
		}
		if _, ok := bySession[live.sessionID]; ok {
			continue
		}
		for i, it := range items {
			if it == nil || used[i] || !itemHasCompleteAgentPair(it) {
				continue
			}
			if agentPairMatches(it, live.agentKind, live.agentSessionID) {
				bySession[live.sessionID] = it
				used[i] = true
				break
			}
		}
	}

	for _, live := range lives {
		if _, ok := bySession[live.sessionID]; ok {
			continue
		}
		for i, it := range items {
			if it == nil || used[i] {
				continue
			}
			if strings.TrimSpace(it.ITermSessionID) == live.sessionID {
				bySession[live.sessionID] = it
				used[i] = true
				break
			}
		}
	}

	var unused []*NoteItem
	for i, it := range items {
		if it == nil || used[i] {
			continue
		}
		if strings.TrimSpace(it.Note) == "" && !it.Bookmarked {
			continue
		}
		unused = append(unused, it)
	}
	return bySession, unused
}

func itemHasCompleteAgentPair(it *NoteItem) bool {
	if it == nil {
		return false
	}
	if strings.TrimSpace(it.AgentRunner) == "" {
		return false
	}
	return itemRunnerID(it) != ""
}

func itemRunnerID(it *NoteItem) string {
	if it == nil {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(it.AgentRunner), "grok") {
		return strings.TrimSpace(it.GrokSessionID)
	}
	return ""
}

func applySnapAgent(ls *LiveSession, agent *kooliterm.SessionAgent) {
	if ls == nil || agent == nil {
		return
	}
	kind := strings.TrimSpace(agent.Kind)
	id := strings.TrimSpace(agent.SessionID)
	ls.agentKind = kind
	ls.agentSessionID = id
	ls.AgentRunner = kind
	if strings.EqualFold(kind, "grok") {
		ls.GrokSessionID = id
	}
}

func liveSessionFromSnap(sess kooliterm.SnapshotSession, win kooliterm.SnapshotWindow, tab kooliterm.SnapshotTab, spaceIdx int, resolveMark func(int) (string, error)) LiveSession {
	winID := ""
	if win.WindowID != 0 {
		winID = strconv.FormatUint(win.WindowID, 10)
	}
	ls := LiveSession{
		SessionID:   strings.TrimSpace(sess.ID),
		SessionName: sess.Name,
		WindowID:    winID,
		WindowName:  win.Name,
		TabIndex:    tab.Index,
		TabName:     tab.Name,
		SpaceIndex:  spaceIdx,
		Desktop:     spaceIdx + 1,
	}
	if sess.Cwd != nil {
		ls.Cwd = *sess.Cwd
	}
	if sess.Idle != nil {
		idle := *sess.Idle
		ls.Idle = &idle
	}
	if sess.PID != nil && *sess.PID > 0 {
		ls.PID = *sess.PID
		applyMarkContent(&ls, resolveMark)
	}
	applySnapAgent(&ls, sess.Agent)
	return ls
}

func applyMarkContent(ls *LiveSession, resolveMark func(int) (string, error)) {
	if ls == nil || ls.PID <= 0 || resolveMark == nil {
		return
	}
	content, err := resolveMark(ls.PID)
	if err != nil || content == "" {
		return
	}
	ls.Mark = content
}

// NeedsAgentEnrich is true when any stored item has a complete agent pair
// (runner + runner id). Inventory capture uses this to decide NoEnrich.
func NeedsAgentEnrich(doc *NotesDocument) bool {
	for _, it := range notesItems(doc) {
		if itemHasCompleteAgentPair(it) {
			return true
		}
	}
	return false
}

func agentPairMatches(it *NoteItem, kind, sessionID string) bool {
	if !itemHasCompleteAgentPair(it) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(it.AgentRunner), strings.TrimSpace(kind)) &&
		itemRunnerID(it) == strings.TrimSpace(sessionID)
}

func orphansFromItems(items []*NoteItem) []OrphanNote {
	var orphans []OrphanNote
	for _, it := range items {
		if it == nil {
			continue
		}
		if strings.TrimSpace(it.Note) == "" && !it.Bookmarked {
			continue
		}
		o := OrphanNote{
			SessionID:  it.ITermSessionID,
			Note:       it.Note,
			Bookmarked: it.Bookmarked,
			UpdatedAt:  it.UpdatedAt,
		}
		if it.LastSeen != nil {
			o.SessionName = it.LastSeen.SessionName
			o.WindowName = it.LastSeen.WindowName
			o.Cwd = it.LastSeen.Cwd
			o.SpaceIndex = it.LastSeen.SpaceIndex
		}
		orphans = append(orphans, o)
	}
	sort.Slice(orphans, func(i, j int) bool {
		if orphans[i].UpdatedAt != orphans[j].UpdatedAt {
			return orphans[i].UpdatedAt > orphans[j].UpdatedAt
		}
		return orphans[i].SessionID < orphans[j].SessionID
	})
	if orphans == nil {
		return []OrphanNote{}
	}
	return orphans
}

// FindLiveSession locates a session UUID in a snapshot.
func FindLiveSession(snap *kooliterm.Snapshot, spaceByWindow map[uint64]int, sessionID string) (LiveSession, bool) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || snap == nil {
		return LiveSession{}, false
	}
	if spaceByWindow == nil {
		spaceByWindow = map[uint64]int{}
	}
	for _, win := range snap.Windows {
		spaceIdx := 0
		if win.WindowID != 0 {
			if idx, ok := spaceByWindow[win.WindowID]; ok && idx >= 0 {
				spaceIdx = idx
			}
		}
		for _, tab := range win.Tabs {
			for _, sess := range tab.Sessions {
				if strings.TrimSpace(sess.ID) != sessionID {
					continue
				}
				return liveSessionFromSnap(sess, win, tab, spaceIdx, nil), true
			}
		}
	}
	return LiveSession{}, false
}

func (h *Handler) handleInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	refresh := strings.TrimSpace(r.URL.Query().Get("refresh")) == "1"
	spaceID, _ := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("space_id")), 10, 64)
	if spaceID != 0 {
		inv, err := h.RefreshSpaceID(spaceID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, inv)
		return
	}
	if !refresh {
		if inv, ok := h.cachedInventory(); ok {
			writeJSON(w, http.StatusOK, inv)
			return
		}
	}
	inv, err := h.waitRefresh()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// RefreshInventory returns the current inventory without HTTP.
// forceFull always deep-captures. Otherwise: last-good (RAM or file) uses
// incremental layout-diff; cold misses use a full capture.
func (h *Handler) RefreshInventory(forceFull bool) (Inventory, error) {
	return h.RefreshInventoryEmit(forceFull, nil)
}

// RefreshInventoryEmit is RefreshInventory plus optional partial publishes
// (each window during a cold/full capture; merge frames when incremental).
func (h *Handler) RefreshInventoryEmit(forceFull bool, emit func(Inventory)) (Inventory, error) {
	if h == nil {
		return Inventory{}, fmt.Errorf("handler is nil")
	}
	if forceFull {
		return h.waitRefreshEmit(emit)
	}
	if _, ok := h.cachedInventory(); ok {
		return h.incrementalRefresh(emit)
	}
	return h.waitRefreshEmit(emit)
}

// CachedInventory returns last-good from RAM or the durable CachePath file
// without probing iTerm (no Layout/Capture). Used for TUI first paint.
func (h *Handler) CachedInventory() (Inventory, bool) {
	return h.cachedInventory()
}

// SeedInventory is the empty desktop scaffold for a cold probing first paint.
func (h *Handler) SeedInventory() Inventory {
	return h.seedInventory(true)
}

func (h *Handler) handleInventoryStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	emit := func(inv Inventory) {
		writeSSE(w, flusher, map[string]any{"type": "inventory", "inventory": inv})
	}

	if cached, ok := h.cachedInventory(); ok {
		// Warm: last-good first. Never emit the empty desktop seed wipe.
		cached.Refreshing = true
		emit(cached)
		inv, err := h.incrementalRefresh(func(partial Inventory) {
			partial.Refreshing = true
			partial.FromCache = false
			emit(partial)
		})
		if err != nil {
			writeSSE(w, flusher, map[string]any{"type": "error", "message": err.Error()})
			return
		}
		inv.Refreshing = false
		inv.FromCache = false
		emit(inv)
		writeSSE(w, flusher, map[string]any{"type": "done"})
		return
	}

	// Cold: desktop headings before any AppleScript, then full deep capture.
	emit(h.seedInventory(true))
	inv, err := h.waitRefreshEmit(func(partial Inventory) {
		partial.Refreshing = true
		emit(partial)
	})
	if err != nil {
		writeSSE(w, flusher, map[string]any{"type": "error", "message": err.Error()})
		return
	}
	inv.Refreshing = false
	emit(inv)
	writeSSE(w, flusher, map[string]any{"type": "done"})
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(b)
	_, _ = w.Write([]byte("\n\n"))
	flusher.Flush()
}

func (h *Handler) cachedInventory() (Inventory, bool) {
	if h == nil {
		return Inventory{}, false
	}
	h.cache.mu.Lock()
	if h.cache.inv != nil {
		out := *h.cache.inv
		out.FromCache = true
		h.cache.mu.Unlock()
		dec := h.decoratePersistHeadings(out)
		dec.FromCache = true
		return dec, true
	}
	h.cache.mu.Unlock()

	// RAM miss: try durable last-good file.
	fileInv, ok := h.loadInventoryCacheFile()
	if !ok {
		return Inventory{}, false
	}
	h.cache.mu.Lock()
	if h.cache.inv != nil {
		// Another caller filled RAM while we read disk.
		out := *h.cache.inv
		out.FromCache = true
		h.cache.mu.Unlock()
		dec := h.decoratePersistHeadings(out)
		dec.FromCache = true
		return dec, true
	}
	cp := fileInv
	cp.FromCache = false
	cp.Refreshing = false
	h.cache.inv = &cp
	out := cp
	out.FromCache = true
	h.cache.mu.Unlock()
	return out, true
}

func (h *Handler) inventoryCachePath() string {
	if h != nil {
		if p := strings.TrimSpace(h.CachePath); p != "" {
			return p
		}
	}
	path, err := DefaultInventoryCachePath()
	if err != nil {
		return ""
	}
	return path
}

// loadInventoryCacheFile reads CachePath. Missing/corrupt/empty-live → miss.
func (h *Handler) loadInventoryCacheFile() (Inventory, bool) {
	path := h.inventoryCachePath()
	if path == "" {
		return Inventory{}, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Inventory{}, false
	}
	var inv Inventory
	if err := json.Unmarshal(b, &inv); err != nil {
		return Inventory{}, false
	}
	if liveCount(inv) < 1 {
		// Never warm from an empty snapshot we may have written by mistake.
		return Inventory{}, false
	}
	inv.FromCache = false
	inv.Refreshing = false
	return h.decoratePersistHeadings(inv), true
}

// persistInventoryCacheFile writes complete last-good Inventory JSON atomically.
// Skips seed/partial (Refreshing) and empty inventories (keeps prior last-good).
func (h *Handler) persistInventoryCacheFile(inv Inventory) {
	if inv.Refreshing {
		return
	}
	if liveCount(inv) < 1 {
		// Includes iterm-down empty: do not overwrite last-good.
		return
	}
	path := h.inventoryCachePath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	// Store API-shaped inventory (not a wrapper). Clear response-only flags.
	toWrite := inv
	toWrite.FromCache = false
	toWrite.Refreshing = false
	data, err := json.MarshalIndent(toWrite, "", "  ")
	if err != nil {
		return
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func (h *Handler) waitRefresh() (Inventory, error) {
	return h.waitRefreshEmit(nil)
}

// waitRefreshEmit runs at most one capture at a time. Concurrent callers join.
// emit is invoked as each iTerm window is collected (leader + subscribers).
func (h *Handler) waitRefreshEmit(emit func(Inventory)) (Inventory, error) {
	if h == nil {
		return Inventory{}, fmt.Errorf("handler is nil")
	}
	h.cache.mu.Lock()
	if emit != nil {
		h.cache.emits = append(h.cache.emits, emit)
	}
	if h.cache.refreshing {
		ch := h.cache.wait
		h.cache.mu.Unlock()
		if ch != nil {
			<-ch
		}
		h.cache.mu.Lock()
		err := h.cache.lastErr
		ok := h.cache.inv != nil
		var out Inventory
		if ok {
			out = *h.cache.inv
			out.FromCache = false
			out.Refreshing = false
		}
		h.cache.mu.Unlock()
		if !ok {
			if err != nil {
				return Inventory{}, err
			}
			return Inventory{}, fmt.Errorf("inventory refresh produced no result")
		}
		return out, err
	}
	ch := make(chan struct{})
	h.cache.refreshing = true
	h.cache.wait = ch
	h.cache.lastErr = nil
	h.cache.mu.Unlock()

	inv, err := h.streamCaptureToCache()

	h.cache.mu.Lock()
	h.cache.lastErr = err
	h.cache.refreshing = false
	h.cache.wait = nil
	h.cache.emits = nil
	close(ch)
	h.cache.mu.Unlock()
	return inv, err
}

func (h *Handler) streamCaptureToCache() (Inventory, error) {
	var last Inventory
	_, err := h.captureStreaming(func(partial *kooliterm.Snapshot) {
		last = h.publishPartial(partial, true)
	})
	if err != nil {
		if isITermNotRunning(err) {
			last = h.publishPartial(nil, false)
			return last, nil
		}
		return last, err
	}
	if last.CachedAt == "" {
		last = h.publishPartial(nil, true)
	}
	last.Refreshing = false
	h.storeCache(last)
	return last, nil
}

// incrementalRefresh is the warm-stream path: Layout-diff, deep-capture new IDs
// only, keep last-good enrich for known IDs, drop gone IDs only on the final frame.
func (h *Handler) incrementalRefresh(emit func(Inventory)) (Inventory, error) {
	last, ok := h.cachedInventory()
	if !ok {
		return h.streamCaptureToCache()
	}
	last.FromCache = false
	floor := liveCount(last)

	if h != nil && h.Layout == nil && h.Capture != nil {
		// Capture injected without Layout (CLI httptest): do not AppleScript.
		// Same-layout increment keeps last-good IDs.
		last.FromCache = false
		last.Refreshing = false
		h.stampCachedAt(&last)
		h.storeCache(last)
		return last, nil
	}
	layout, err := h.layoutFn()()
	if err != nil {
		if isITermNotRunning(err) {
			out := h.publishPartial(nil, false)
			out.Refreshing = false
			return out, nil
		}
		// Keep last-good rather than failing the warm stream.
		last.FromCache = false
		last.Refreshing = false
		h.storeCache(last)
		return last, nil
	}

	var captured *kooliterm.Snapshot
	if hasNewSessionIDs(last, layout) {
		snap, capErr := h.captureFn()()
		if capErr != nil {
			if isITermNotRunning(capErr) {
				out := h.publishPartial(nil, false)
				out.Refreshing = false
				return out, nil
			}
			return last, capErr
		}
		captured = snap
	}

	if emit != nil {
		keep := h.mergeLayout(last, layout, captured, false)
		if liveCount(keep) >= floor {
			keep.Refreshing = true
			keep.FromCache = false
			emit(keep)
		}
	}

	final := h.mergeLayout(last, layout, captured, true)
	final.Refreshing = false
	final.FromCache = false
	h.stampCachedAt(&final)
	h.storeCache(final)
	return final, nil
}

func (h *Handler) mergeLayout(last Inventory, layout, captured *kooliterm.Snapshot, dropGone bool) Inventory {
	running := last.ITermRunning
	n, _ := h.countDesktopsFn()()
	notes := h.noteStore().Document()
	desktops := desktopsFromCount(n)

	layoutInv := h.assembleInventory(layout, desktops, h.resolveSpaces(layout), notes, running)
	var capByID map[string]LiveSession
	if captured != nil {
		capInv := h.assembleInventory(captured, desktops, h.resolveSpaces(captured), notes, running)
		capByID = liveByID(capInv)
	}
	lastByID := liveByID(last)

	var lives []LiveSession
	seen := make(map[string]struct{})
	for _, s := range liveSessions(layoutInv) {
		seen[s.SessionID] = struct{}{}
		if prev, ok := lastByID[s.SessionID]; ok {
			// Keep last-good enrich, but Space can change when Desktops are deleted/reordered.
			prev.SpaceIndex = s.SpaceIndex
			prev.Desktop = s.Desktop
			lives = append(lives, prev)
			continue
		}
		if cap, ok := capByID[s.SessionID]; ok {
			lives = append(lives, cap)
			continue
		}
		lives = append(lives, s)
	}
	if !dropGone {
		for _, prev := range liveSessions(last) {
			if _, ok := seen[prev.SessionID]; !ok {
				lives = append(lives, prev)
			}
		}
	}
	return h.inventoryFromSessions(lives, running)
}

func (h *Handler) inventoryFromSessions(lives []LiveSession, running bool) Inventory {
	n, _ := h.countDesktopsFn()()
	notes := h.noteStore().Document()
	inv := BuildInventory(nil, desktopsFromCount(n), nil, notes, running)
	bySpace := map[int][]LiveSession{}
	maxSpace := n - 1
	if maxSpace < 0 {
		maxSpace = 0
	}
	for _, s := range lives {
		if s.SpaceIndex < 0 || s.SpaceIndex > maxSpace {
			// Stale last-good indexes from deleted Desktops must not grow the list.
			continue
		}
		bySpace[s.SpaceIndex] = append(bySpace[s.SpaceIndex], s)
	}
	have := make(map[int]int, len(inv.Desktops))
	for i, d := range inv.Desktops {
		have[d.SpaceIndex] = i
	}
	for space := 0; space <= maxSpace; space++ {
		if _, ok := have[space]; ok {
			continue
		}
		inv.Desktops = append(inv.Desktops, DesktopGroup{
			SpaceIndex: space,
			Desktop:    space + 1,
			Sessions:   []LiveSession{},
		})
		have[space] = len(inv.Desktops) - 1
	}
	sort.Slice(inv.Desktops, func(i, j int) bool {
		return inv.Desktops[i].SpaceIndex < inv.Desktops[j].SpaceIndex
	})
	for i := range inv.Desktops {
		sess := bySpace[inv.Desktops[i].SpaceIndex]
		if sess == nil {
			sess = []LiveSession{}
		}
		inv.Desktops[i].Sessions = sess
	}
	inv = ApplyNotes(inv, notes)
	if spaces, err := h.listSpacesFn()(); err == nil {
		inv = ClipDesktops(inv, spaces)
	}
	return h.decorateInventory(inv)
}

func (h *Handler) stampCachedAt(inv *Inventory) {
	if inv == nil {
		return
	}
	now := time.Now()
	if h != nil && h.Now != nil {
		now = h.Now()
	}
	inv.CachedAt = now.UTC().Format(time.RFC3339)
}

func liveSessions(inv Inventory) []LiveSession {
	var out []LiveSession
	for _, d := range inv.Desktops {
		out = append(out, d.Sessions...)
	}
	return out
}

func liveCount(inv Inventory) int {
	n := 0
	for _, d := range inv.Desktops {
		n += len(d.Sessions)
	}
	return n
}

func liveByID(inv Inventory) map[string]LiveSession {
	out := make(map[string]LiveSession)
	for _, s := range liveSessions(inv) {
		out[s.SessionID] = s
	}
	return out
}

func sessionIDsFromSnap(snap *kooliterm.Snapshot) map[string]struct{} {
	out := map[string]struct{}{}
	if snap == nil {
		return out
	}
	for _, w := range snap.Windows {
		for _, tab := range w.Tabs {
			for _, s := range tab.Sessions {
				id := strings.TrimSpace(s.ID)
				if id == "" {
					continue
				}
				out[id] = struct{}{}
			}
		}
	}
	return out
}

func hasNewSessionIDs(last Inventory, layout *kooliterm.Snapshot) bool {
	have := make(map[string]struct{})
	for _, s := range liveSessions(last) {
		have[s.SessionID] = struct{}{}
	}
	for id := range sessionIDsFromSnap(layout) {
		if _, ok := have[id]; !ok {
			return true
		}
	}
	return false
}

func (h *Handler) publishPartial(snap *kooliterm.Snapshot, running bool) Inventory {
	n, _ := h.countDesktopsFn()()
	inv := h.assembleInventory(snap, desktopsFromCount(n), h.resolveSpaces(snap), h.noteStore().Document(), running)
	now := time.Now()
	if h != nil && h.Now != nil {
		now = h.Now()
	}
	inv.CachedAt = now.UTC().Format(time.RFC3339)
	inv.FromCache = false
	inv.Refreshing = true
	inv = h.decorateInventory(inv)
	h.storeCache(inv)
	h.cache.mu.Lock()
	cbs := make([]func(Inventory), len(h.cache.emits))
	copy(cbs, h.cache.emits)
	h.cache.mu.Unlock()
	for _, cb := range cbs {
		if cb != nil {
			cb(inv)
		}
	}
	return inv
}

func (h *Handler) storeCache(inv Inventory) {
	if h == nil {
		return
	}
	h.cache.mu.Lock()
	cp := inv
	h.cache.inv = &cp
	h.cache.mu.Unlock()
	// Always update RAM; disk only for complete last-good with live sessions.
	h.persistInventoryCacheFile(inv)
}

func (h *Handler) seedInventory(refreshing bool) Inventory {
	n, _ := h.countDesktopsFn()()
	running := true
	if h != nil && h.ITermRunning != nil {
		running = h.ITermRunning()
	}
	inv := BuildInventory(nil, desktopsFromCount(n), nil, h.noteStore().Document(), running)
	inv.Refreshing = refreshing
	inv.FromCache = false
	return h.decorateInventory(inv)
}

func (h *Handler) captureStreaming(onPartial func(*kooliterm.Snapshot)) (*kooliterm.Snapshot, error) {
	var acc []kooliterm.SnapshotWindow
	onWin := func(win kooliterm.SnapshotWindow) error {
		acc = append(acc, win)
		if onPartial != nil {
			onPartial(&kooliterm.Snapshot{Windows: append([]kooliterm.SnapshotWindow(nil), acc...)})
		}
		return nil
	}
	if h != nil && h.CaptureStream != nil {
		return h.CaptureStream(onWin)
	}
	if h != nil && h.Capture != nil {
		snap, err := h.Capture()
		if err != nil {
			return nil, err
		}
		if snap != nil {
			for _, w := range snap.Windows {
				if err := onWin(w); err != nil {
					return snap, err
				}
			}
		}
		return snap, nil
	}
	snap, _, err := kooliterm.CaptureSnapshotForSaveStream(h.captureOpts(), onWin)
	return snap, err
}

func (h *Handler) captureOpts() kooliterm.CaptureOpts {
	var doc *NotesDocument
	if h != nil {
		doc = h.noteStore().Document()
	}
	return kooliterm.CaptureOpts{NoEnrich: !NeedsAgentEnrich(doc)}
}

func (h *Handler) captureSpace(spaceIndex int) (*kooliterm.Snapshot, error) {
	if h != nil && h.CaptureSpace != nil {
		return h.CaptureSpace(spaceIndex)
	}
	if h != nil && h.Capture != nil {
		return h.Capture()
	}
	opts := h.captureOpts()
	opts.SpaceAllow = []int{spaceIndex}
	snap, _, err := kooliterm.CaptureSnapshotWith(opts)
	return snap, err
}

func (h *Handler) spaceIndexForID(spaceID uint64) (int, bool) {
	if spaceID == 0 {
		return 0, false
	}
	if spaces, err := h.listSpacesFn()(); err == nil {
		for _, s := range spaces {
			if s.SpaceID == spaceID {
				return s.SpaceIndex, true
			}
		}
	}
	if last, ok := h.cachedInventory(); ok {
		for _, d := range last.Desktops {
			if d.SpaceID == spaceID {
				return d.SpaceIndex, true
			}
		}
	}
	return 0, false
}

// RefreshSpaceID recaptures one Desktop by CGS id and merges it into last-good.
// Unknown ids or a cold cache fall through to a full refresh.
func (h *Handler) RefreshSpaceID(spaceID uint64) (Inventory, error) {
	idx, ok := h.spaceIndexForID(spaceID)
	if !ok {
		return h.waitRefresh()
	}
	return h.refreshSpace(idx)
}

func (h *Handler) refreshSpace(spaceIndex int) (Inventory, error) {
	last, ok := h.cachedInventory()
	if !ok {
		return h.waitRefresh()
	}
	snap, err := h.captureSpace(spaceIndex)
	if err != nil {
		if isITermNotRunning(err) {
			out := h.publishPartial(nil, false)
			out.Refreshing = false
			return out, nil
		}
		return last, err
	}
	n, _ := h.countDesktopsFn()()
	running := last.ITermRunning
	if h != nil && h.ITermRunning != nil {
		running = h.ITermRunning()
	}
	capInv := h.assembleInventory(snap, desktopsFromCount(n), h.resolveSpaces(snap), h.noteStore().Document(), running)
	out := mergeSpaceSessions(last, spaceIndex, capInv)
	out = ApplyNotes(out, h.noteStore().Document())
	out.Refreshing = false
	out.FromCache = false
	h.stampCachedAt(&out)
	out = h.decorateInventory(out)
	h.storeCache(out)
	return out, nil
}

// mergeSpaceSessions replaces last-good sessions on spaceIndex with cap's.
func mergeSpaceSessions(last Inventory, spaceIndex int, cap Inventory) Inventory {
	sess := []LiveSession{}
	for _, d := range cap.Desktops {
		if d.SpaceIndex == spaceIndex {
			if d.Sessions != nil {
				sess = d.Sessions
			}
			break
		}
	}
	out := last
	desktops := make([]DesktopGroup, len(last.Desktops))
	copy(desktops, last.Desktops)
	found := false
	for i := range desktops {
		if desktops[i].SpaceIndex == spaceIndex {
			desktops[i].Sessions = sess
			found = true
		}
	}
	if !found {
		desktops = append(desktops, DesktopGroup{
			SpaceIndex: spaceIndex,
			Desktop:    spaceIndex + 1,
			Sessions:   sess,
		})
		sort.Slice(desktops, func(i, j int) bool {
			return desktops[i].SpaceIndex < desktops[j].SpaceIndex
		})
	}
	out.Desktops = desktops
	return out
}

// desktopsFromCount synthesizes Desktop 1..n without opening Mission Control.
func desktopsFromCount(n int) []spacelib.Desktop {
	if n < 1 {
		return nil
	}
	out := make([]spacelib.Desktop, n)
	for i := 0; i < n; i++ {
		out[i] = spacelib.Desktop{Number: i + 1, Name: fmt.Sprintf("Desktop %d", i+1)}
	}
	return out
}

func (h *Handler) resolveSpaces(snap *kooliterm.Snapshot) map[uint64]int {
	out := map[uint64]int{}
	if snap == nil {
		return out
	}
	fn := h.spaceForWindowFn()
	for _, win := range snap.Windows {
		if win.WindowID == 0 {
			continue
		}
		idx, err := fn(win.WindowID)
		if err != nil || idx < 0 {
			out[win.WindowID] = 0
			continue
		}
		out[win.WindowID] = idx
	}
	return out
}

func isITermNotRunning(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not running")
}
