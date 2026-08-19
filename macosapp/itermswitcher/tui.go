// Package itermswitcher — pure terminals list TUI state (no stdout / TTY / AppleScript).
package itermswitcher

import (
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xhd2015/ai-critic/server/localiterm2"
)

// Focus pane ids for the inline split TUI.
const (
	PaneList    = "list"
	PaneSidebar = "sidebar"
)

// Status tokens painted in the chrome (locked UX).
const (
	StatusCached      = "cached"
	StatusIncremental = "incremental…"
	StatusProbing     = "probing…"
	StatusUpToDate    = "up to date"
)

// UIAction is the outcome of ApplyKey (empty Name = stay open, no side effect).
type UIAction struct {
	// Name is "focus", "quit", or empty.
	Name string
	// SessionID is set when Name == "focus".
	SessionID string
}

// UIState is pure two-pane state for terminals list.
type UIState struct {
	Inventory    localiterm2.Inventory
	FocusPane    string // PaneList (default) or PaneSidebar
	SidebarIndex int    // selected sidebar row
	ListIndex    int    // selected session row in the filtered list
	Status       string // StatusCached / StatusIncremental / …
	// SidebarIDs is the ordered sidebar filter ids (all, bookmarks, desktop:N, …).
	SidebarIDs []string
}

// NewUIState builds initial state from a complete inventory.
// Default focus is the list; Status is typically StatusCached for file first paint.
func NewUIState(inv localiterm2.Inventory, status string) UIState {
	if status == "" {
		status = StatusCached
	}
	return UIState{
		Inventory:    inv,
		FocusPane:    PaneList,
		SidebarIndex: 0,
		ListIndex:    0,
		Status:       status,
		SidebarIDs:   sidebarIDsFromInv(inv),
	}
}

func sidebarIDsFromInv(inv localiterm2.Inventory) []string {
	spaces := map[int]struct{}{}
	for _, d := range inv.Desktops {
		spaces[d.SpaceIndex] = struct{}{}
	}
	idxs := make([]int, 0, len(spaces))
	for i := range spaces {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	bookmarks := 0
	for _, s := range liveFilterSessions(inv) {
		if s.Bookmarked {
			bookmarks++
		}
	}
	items := SidebarItems(idxs, bookmarks, len(inv.SavedNotes))
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

func liveFilterSessions(inv localiterm2.Inventory) []FilterSession {
	var out []FilterSession
	for _, d := range inv.Desktops {
		for _, s := range d.Sessions {
			out = append(out, FilterSession{
				Name:       s.SessionName,
				Note:       s.Note,
				Cwd:        s.Cwd,
				WindowName: s.WindowName,
				TabName:    s.TabName,
				SessionID:  s.SessionID,
				SpaceIndex: s.SpaceIndex,
				SpaceLabel: d.Label,
				Bookmarked: s.Bookmarked,
			})
		}
	}
	return out
}

func desktopLabel(inv localiterm2.Inventory, spaceIndex int) string {
	for _, d := range inv.Desktops {
		if d.SpaceIndex == spaceIndex {
			return d.Label
		}
	}
	return ""
}

type listRow struct {
	primary   string
	sessionID string
	skip      bool
}

func currentSidebarID(s UIState) string {
	if len(s.SidebarIDs) == 0 {
		return SidebarAll
	}
	if s.SidebarIndex < 0 {
		return s.SidebarIDs[0]
	}
	if s.SidebarIndex >= len(s.SidebarIDs) {
		return s.SidebarIDs[len(s.SidebarIDs)-1]
	}
	return s.SidebarIDs[s.SidebarIndex]
}

func listRows(s UIState) []listRow {
	id := currentSidebarID(s)
	if id == SidebarSaved {
		var rows []listRow
		for _, o := range s.Inventory.SavedNotes {
			rows = append(rows, listRow{
				primary:   FormatOrphanPrimary(o.Note, o.SessionName, o.Cwd, o.SessionID),
				sessionID: o.SessionID,
			})
		}
		return rows
	}
	if idx, ok := ParseSidebarDesktop(id); ok {
		rows := []listRow{
			{primary: FormatSpaceLabelRow(desktopLabel(s.Inventory, idx))},
			{primary: FormatSpaceLabelSeparator(), skip: true},
		}
		filtered := FilterSessions(liveFilterSessions(s.Inventory), id, "")
		for _, fs := range filtered {
			rows = append(rows, listRow{
				primary:   FormatSessionPrimary(fs.Name, fs.Cwd, fs.SessionID),
				sessionID: fs.SessionID,
			})
		}
		return rows
	}
	filtered := FilterSessions(liveFilterSessions(s.Inventory), id, "")
	rows := make([]listRow, 0, len(filtered))
	for _, fs := range filtered {
		rows = append(rows, listRow{
			primary:   FormatSessionPrimary(fs.Name, fs.Cwd, fs.SessionID),
			sessionID: fs.SessionID,
		})
	}
	return rows
}

func firstSelectableIndex(rows []listRow) int {
	for i, r := range rows {
		if !r.skip {
			return i
		}
	}
	return 0
}

func clampListIndex(s UIState) UIState {
	rows := listRows(s)
	n := len(rows)
	if n == 0 {
		s.ListIndex = 0
		return s
	}
	if s.ListIndex < 0 {
		s.ListIndex = 0
	}
	if s.ListIndex >= n {
		s.ListIndex = n - 1
	}
	if rows[s.ListIndex].skip {
		for i := s.ListIndex + 1; i < n; i++ {
			if !rows[i].skip {
				s.ListIndex = i
				return s
			}
		}
		for i := s.ListIndex - 1; i >= 0; i-- {
			if !rows[i].skip {
				s.ListIndex = i
				return s
			}
		}
		s.ListIndex = firstSelectableIndex(rows)
	}
	return s
}

func moveList(s UIState, dir int) UIState {
	rows := listRows(s)
	n := len(rows)
	if n == 0 {
		return s
	}
	i := s.ListIndex + dir
	for i >= 0 && i < n {
		if !rows[i].skip {
			s.ListIndex = i
			return s
		}
		i += dir
	}
	return s
}

func clampSidebarIndex(s UIState) UIState {
	n := len(s.SidebarIDs)
	if n == 0 {
		s.SidebarIndex = 0
		return s
	}
	if s.SidebarIndex < 0 {
		s.SidebarIndex = 0
	}
	if s.SidebarIndex >= n {
		s.SidebarIndex = n - 1
	}
	return s
}

// View renders the inline split box (no alt-screen) as screen lines.
func View(s UIState) []string {
	if len(s.SidebarIDs) == 0 {
		s.SidebarIDs = sidebarIDsFromInv(s.Inventory)
	}
	s = clampSidebarIndex(s)
	s = clampListIndex(s)

	const leftW = 14
	const rightW = 45

	side := make([]string, len(s.SidebarIDs))
	for i, id := range s.SidebarIDs {
		marker := "  "
		if s.FocusPane == PaneSidebar && i == s.SidebarIndex {
			marker = "› "
		}
		title := FormatSidebarTitle(id)
		if idx, ok := ParseSidebarDesktop(id); ok {
			title = FormatSidebarDesktopTitle(idx, desktopLabel(s.Inventory, idx))
		}
		side[i] = marker + title
	}
	rows := listRows(s)
	list := make([]string, len(rows))
	for i, r := range rows {
		marker := "  "
		if s.FocusPane == PaneList && i == s.ListIndex && !r.skip {
			marker = "› "
		}
		list[i] = marker + r.primary
	}

	n := len(side)
	if len(list) > n {
		n = len(list)
	}
	if n < 1 {
		n = 1
	}
	for len(side) < n {
		side = append(side, "")
	}
	for len(list) < n {
		list = append(list, "")
	}

	// Title: " Terminals … status"
	innerW := leftW + rightW + 1 // middle border column counted in outer frame only
	_ = innerW
	frameW := leftW + rightW + 3 // │ left │ right │
	title := " " + FormatWindowTitle()
	status := s.Status
	if status == "" {
		status = StatusCached
	}
	pad := frameW - utf8.RuneCountInString(title) - utf8.RuneCountInString(status) - 1
	if pad < 1 {
		pad = 1
	}
	lines := make([]string, 0, n+3)
	lines = append(lines, title+strings.Repeat(" ", pad)+status)
	lines = append(lines, "┌"+strings.Repeat("─", leftW)+"┬"+strings.Repeat("─", rightW)+"┐")
	for i := 0; i < n; i++ {
		lines = append(lines, "│"+fitWidth(side[i], leftW)+"│"+fitWidth(list[i], rightW)+"│")
	}
	lines = append(lines, "└"+strings.Repeat("─", leftW)+"┴"+strings.Repeat("─", rightW)+"┘")
	lines = append(lines, " ↑↓ move   ←→ pane   ⏎ focus   q quit")
	return lines
}

func fitWidth(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		return string(r[:w])
	}
	return string(r) + strings.Repeat(" ", w-len(r))
}

// ApplyKey updates state for one key. Key names: j/k/up/down, h/l/tab/left/right,
// enter, q, esc, ], [, "1"–"9".
func ApplyKey(s UIState, key string) (UIState, UIAction) {
	if len(s.SidebarIDs) == 0 {
		s.SidebarIDs = sidebarIDsFromInv(s.Inventory)
	}
	s = clampSidebarIndex(s)
	s = clampListIndex(s)

	key = strings.ToLower(strings.TrimSpace(key))
	switch key {
	case "q", "esc":
		return s, UIAction{Name: "quit"}
	case "enter":
		rows := listRows(s)
		if s.ListIndex >= 0 && s.ListIndex < len(rows) {
			id := strings.TrimSpace(rows[s.ListIndex].sessionID)
			if id != "" {
				return s, UIAction{Name: "focus", SessionID: id}
			}
		}
		return s, UIAction{}
	case "j", "down":
		if s.FocusPane == PaneSidebar {
			if s.SidebarIndex < len(s.SidebarIDs)-1 {
				s.SidebarIndex++
			}
			s.ListIndex = 0
			return clampListIndex(s), UIAction{}
		}
		return moveList(s, +1), UIAction{}
	case "k", "up":
		if s.FocusPane == PaneSidebar {
			if s.SidebarIndex > 0 {
				s.SidebarIndex--
			}
			s.ListIndex = 0
			return clampListIndex(s), UIAction{}
		}
		return moveList(s, -1), UIAction{}
	case "tab":
		if s.FocusPane == PaneList {
			s.FocusPane = PaneSidebar
		} else {
			s.FocusPane = PaneList
		}
		return s, UIAction{}
	case "l", "right":
		s.FocusPane = PaneSidebar
		return s, UIAction{}
	case "h", "left":
		s.FocusPane = PaneList
		return s, UIAction{}
	case "]":
		return advanceDesktop(s, +1), UIAction{}
	case "[":
		return advanceDesktop(s, -1), UIAction{}
	default:
		if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			n, _ := strconv.Atoi(key)
			return jumpDesktop(s, n), UIAction{}
		}
	}
	return s, UIAction{}
}

func advanceDesktop(s UIState, dir int) UIState {
	s.FocusPane = PaneList
	var deskPos []int
	for i, id := range s.SidebarIDs {
		if _, ok := ParseSidebarDesktop(id); ok {
			deskPos = append(deskPos, i)
		}
	}
	if len(deskPos) == 0 {
		return s
	}
	curID := currentSidebarID(s)
	_, onDesk := ParseSidebarDesktop(curID)
	if !onDesk {
		if dir > 0 {
			s.SidebarIndex = deskPos[0]
		} else {
			s.SidebarIndex = deskPos[len(deskPos)-1]
		}
		s.ListIndex = 0
		return clampListIndex(s)
	}
	pos := 0
	for i, di := range deskPos {
		if di == s.SidebarIndex {
			pos = i
			break
		}
	}
	pos += dir
	if pos < 0 {
		pos = 0
	}
	if pos >= len(deskPos) {
		pos = len(deskPos) - 1
	}
	s.SidebarIndex = deskPos[pos]
	s.ListIndex = 0
	return clampListIndex(s)
}

func jumpDesktop(s UIState, desktop1Based int) UIState {
	if desktop1Based < 1 {
		return s
	}
	want := SidebarDesktopID(desktop1Based - 1)
	for i, id := range s.SidebarIDs {
		if id == want {
			s.SidebarIndex = i
			s.FocusPane = PaneList
			s.ListIndex = 0
			return clampListIndex(s)
		}
	}
	return s
}

// WithInventory replaces the inventory (partial probe or increment) without
// resetting pane focus. List/sidebar indexes are clamped to the new rows.
func WithInventory(s UIState, inv localiterm2.Inventory, status string) UIState {
	s.Inventory = inv
	if status != "" {
		s.Status = status
	}
	s.SidebarIDs = sidebarIDsFromInv(inv)
	return clampListIndex(clampSidebarIndex(s))
}

// PaintView joins View lines for writing to a terminal/buffer (trailing newline).
func PaintView(s UIState) string {
	lines := View(s)
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
