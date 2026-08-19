package itermswitcher

import (
	"strconv"
	"strings"
)

const (
	// SidebarAll lists every live session.
	SidebarAll = "all"
	// SidebarBookmarks lists live starred sessions.
	SidebarBookmarks = "bookmarks"
	// SidebarSaved lists orphan notes / gone bookmarks.
	SidebarSaved         = "saved"
	sidebarDesktopPrefix = "desktop:"
)

// SidebarItem is one row in the switcher sidebar.
type SidebarItem struct {
	ID    string
	Title string
	Count int
}

// FilterSession is the subset of a live row needed to filter.
type FilterSession struct {
	Name       string
	Note       string
	Cwd        string
	WindowName string
	TabName    string
	SessionID  string
	SpaceIndex int
	SpaceLabel string
	Bookmarked bool
}

// FormatSidebarTitle maps a sidebar id to its label.
// unknown ids return empty. "desktop:N" uses FormatDesktopHeader.
func FormatSidebarTitle(id string) string {
	switch strings.TrimSpace(id) {
	case "", SidebarAll:
		return "All"
	case SidebarBookmarks:
		return "Bookmarks"
	case SidebarSaved:
		return "Saved notes"
	}
	if idx, ok := ParseSidebarDesktop(id); ok {
		return FormatDesktopHeader(idx)
	}
	return ""
}

// FormatBookmarksHeader is the Bookmarks sidebar label.
func FormatBookmarksHeader() string {
	return FormatSidebarTitle(SidebarBookmarks)
}

// SidebarDesktopID returns "desktop:N" for a 0-based space index.
func SidebarDesktopID(spaceIndex int) string {
	if spaceIndex < 0 {
		spaceIndex = 0
	}
	return sidebarDesktopPrefix + strconv.Itoa(spaceIndex)
}

// ParseSidebarDesktop parses "desktop:N". N must be >= 0.
func ParseSidebarDesktop(id string) (int, bool) {
	id = strings.TrimSpace(id)
	if !strings.HasPrefix(id, sidebarDesktopPrefix) {
		return 0, false
	}
	n, err := strconv.Atoi(id[len(sidebarDesktopPrefix):])
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// MatchesSidebar reports whether a live session belongs on the sidebar page.
// Saved notes never match live rows. Unknown ids behave like All.
func MatchesSidebar(sidebarID string, spaceIndex int, bookmarked bool) bool {
	id := strings.TrimSpace(sidebarID)
	switch id {
	case "", SidebarAll:
		return true
	case SidebarBookmarks:
		return bookmarked
	case SidebarSaved:
		return false
	}
	if idx, ok := ParseSidebarDesktop(id); ok {
		if spaceIndex < 0 {
			spaceIndex = 0
		}
		return idx == spaceIndex
	}
	return true
}

// FilterSessions keeps live rows that match sidebar + query.
func FilterSessions(sessions []FilterSession, sidebarID, query string) []FilterSession {
	out := make([]FilterSession, 0, len(sessions))
	for _, s := range sessions {
		if !MatchesSidebar(sidebarID, s.SpaceIndex, s.Bookmarked) {
			continue
		}
		if !SessionMatches(s.Name, s.Note, s.Cwd, s.WindowName, s.TabName, s.SessionID, s.SpaceIndex, s.SpaceLabel, query) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// CountBookmarked returns how many live rows are starred.
func CountBookmarked(sessions []FilterSession) int {
	n := 0
	for _, s := range sessions {
		if s.Bookmarked {
			n++
		}
	}
	return n
}

// SidebarItems builds All, Bookmarks, each Desktop, Saved notes.
func SidebarItems(spaceIndexes []int, bookmarkCount, savedCount int) []SidebarItem {
	items := []SidebarItem{
		{ID: SidebarAll, Title: FormatSidebarTitle(SidebarAll)},
		{ID: SidebarBookmarks, Title: FormatSidebarTitle(SidebarBookmarks), Count: bookmarkCount},
	}
	for _, idx := range spaceIndexes {
		id := SidebarDesktopID(idx)
		items = append(items, SidebarItem{ID: id, Title: FormatSidebarTitle(id)})
	}
	items = append(items, SidebarItem{
		ID:    SidebarSaved,
		Title: FormatSidebarTitle(SidebarSaved),
		Count: savedCount,
	})
	return items
}

// FormatOrphanPrimary prefers the note, else last-seen name/cwd/id.
func FormatOrphanPrimary(note, sessionName, cwd, sessionID string) string {
	if s := strings.TrimSpace(note); s != "" {
		return s
	}
	return FormatSessionPrimary(sessionName, cwd, sessionID)
}

// FormatWindowTitle is the switcher panel title.
func FormatWindowTitle() string {
	return "Terminals"
}

// ResolvedBookmarked prefers a local override (optimistic star) over inventory.
func ResolvedBookmarked(sessionID string, inventoryValue bool, overrides map[string]bool) bool {
	if overrides != nil {
		if v, ok := overrides[sessionID]; ok {
			return v
		}
	}
	return inventoryValue
}

// ReconcileBookmarkOverrides drops overrides that match live inventory (or vanished).
func ReconcileBookmarkOverrides(overrides map[string]bool, live map[string]bool) map[string]bool {
	if len(overrides) == 0 {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(overrides))
	for id, want := range overrides {
		cur, ok := live[id]
		if !ok || cur == want {
			continue
		}
		out[id] = want
	}
	return out
}
