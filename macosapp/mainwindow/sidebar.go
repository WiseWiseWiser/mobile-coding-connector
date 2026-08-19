// Package mainwindow holds pure labels for the menu-bar "Show App" window.
package mainwindow

// StorageKey is the UserDefaults / @AppStorage key for the last sidebar page.
const StorageKey = "mainSidebarPage"

// DefaultSidebarID is the first-launch page.
const DefaultSidebarID = "home"

// SidebarIDs is the v1 sidebar order.
func SidebarIDs() []string {
	return []string{"home", "services", "projects", "settings"}
}

// FormatSidebarTitle maps a sidebar id to its display title.
func FormatSidebarTitle(id string) string {
	switch id {
	case "home":
		return "Home"
	case "services":
		return "Services"
	case "projects":
		return "Projects"
	case "settings":
		return "Settings"
	default:
		return ""
	}
}

// FormatShowAppLabel is the menu-bar item that fronts the main window.
func FormatShowAppLabel() string {
	return "Show App"
}

// NormalizeSidebarID returns id if known, else DefaultSidebarID.
func NormalizeSidebarID(id string) string {
	for _, known := range SidebarIDs() {
		if id == known {
			return id
		}
	}
	return DefaultSidebarID
}
