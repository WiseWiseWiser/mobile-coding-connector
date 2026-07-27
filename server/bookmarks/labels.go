package bookmarks

// FormatEmptyBookmarksLabel is the menu-bar copy when the bookmarks tree has no items.
func FormatEmptyBookmarksLabel() string {
	return "No bookmarks"
}

// FormatBookmarkMenuTitle returns the display title for a bookmark menu item.
func FormatBookmarkMenuTitle(name string) string {
	return name
}
