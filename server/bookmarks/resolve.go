package bookmarks

import "strings"

// ResolveBrowser returns the effective browser preference for opening a bookmark.
// Non-empty bookmarkBrowser wins; otherwise globalDefault; if both empty → "default".
func ResolveBrowser(bookmarkBrowser, globalDefault string) string {
	if b := strings.TrimSpace(bookmarkBrowser); b != "" {
		return b
	}
	if g := strings.TrimSpace(globalDefault); g != "" {
		return g
	}
	return "default"
}
