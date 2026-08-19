// Package itermswitcher holds pure display helpers for the local iTerm switcher.
package itermswitcher

import (
	"fmt"
	"strings"
)

// FormatSessionPrimary is the first-column title: name, else cwd, else short id.
func FormatSessionPrimary(name, cwd, sessionID string) string {
	if s := strings.TrimSpace(name); s != "" {
		return s
	}
	if s := strings.TrimSpace(cwd); s != "" {
		return s
	}
	return ShortSessionID(sessionID)
}

// FormatSessionNote is the note column; empty → em dash.
func FormatSessionNote(note string) string {
	if s := strings.TrimSpace(note); s != "" {
		return s
	}
	return "—"
}

// FormatSessionTitle composes "primary  ·  note" for compact rows.
func FormatSessionTitle(name, note, cwd, sessionID string) string {
	primary := FormatSessionPrimary(name, cwd, sessionID)
	n := FormatSessionNote(note)
	if n == "—" {
		return primary
	}
	return primary + "  ·  " + n
}

// FormatDesktopHeader returns "Desktop N" for a 0-based space index.
func FormatDesktopHeader(spaceIndex int) string {
	if spaceIndex < 0 {
		spaceIndex = 0
	}
	return fmt.Sprintf("Desktop %d", spaceIndex+1)
}

// FormatSidebarDesktopTitle prefers a user Space label, else "Desktop N".
func FormatSidebarDesktopTitle(spaceIndex int, label string) string {
	if s := strings.TrimSpace(label); s != "" {
		return s
	}
	return FormatDesktopHeader(spaceIndex)
}

// FormatSpaceLabelRow is the Per Space action row: prompt or current label.
func FormatSpaceLabelRow(label string) string {
	if s := strings.TrimSpace(label); s != "" {
		return s
	}
	return "Set Space Label"
}

// FormatChangeSpaceLabel is the Change button title.
func FormatChangeSpaceLabel() string {
	return "Change"
}

// FormatClearSpaceLabel is the Clear button title.
func FormatClearSpaceLabel() string {
	return "Clear"
}

// FormatSpaceLabelSeparator is the TUI rule between the label row and sessions.
func FormatSpaceLabelSeparator() string {
	return "────────"
}

// FormatEmptyITerm is the empty-state copy when iTerm is not running.
func FormatEmptyITerm() string {
	return "iTerm2 is not running"
}

// FormatSavedNotesHeader returns "Saved notes (N)".
func FormatSavedNotesHeader(n int) string {
	if n < 0 {
		n = 0
	}
	return fmt.Sprintf("Saved notes (%d)", n)
}

// FormatBusyLabel maps idle pointer to busy/idle/empty.
func FormatBusyLabel(idle *bool) string {
	if idle == nil {
		return ""
	}
	if *idle {
		return "idle"
	}
	return "busy"
}

// FormatSessionSubtitle is "cwd  ·  tab N  ·  busy".
func FormatSessionSubtitle(cwd string, tabIndex int, idle *bool) string {
	var parts []string
	if s := strings.TrimSpace(cwd); s != "" {
		parts = append(parts, s)
	}
	if tabIndex > 0 {
		parts = append(parts, fmt.Sprintf("tab %d", tabIndex))
	}
	if b := FormatBusyLabel(idle); b != "" {
		parts = append(parts, b)
	}
	return strings.Join(parts, "  ·  ")
}

// ShortSessionID returns the first 8 hex chars of a UUID, or the id itself.
func ShortSessionID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if i := strings.IndexByte(id, '-'); i > 0 {
		return id[:i]
	}
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// FormatDefaultHotKey is the default ⌘⇧Space label.
func FormatDefaultHotKey() string {
	return "⌘⇧Space"
}

// SessionMatches reports whether query is a substring of session fields.
// Empty/whitespace query matches everything. Comparison is case-insensitive.
func SessionMatches(name, note, cwd, windowName, tabName, sessionID string, spaceIndex int, spaceLabel, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	desktop := strings.ToLower(FormatSidebarDesktopTitle(spaceIndex, spaceLabel))
	hay := strings.ToLower(strings.Join([]string{
		name, note, cwd, windowName, tabName, desktop, spaceLabel, sessionID,
	}, " "))
	return strings.Contains(hay, q)
}
