package browser

// FormatOpenInBrowserLabel maps a stored browser preference to the menu-bar
// label for the Open in Browser action.
func FormatOpenInBrowserLabel(browser string) string {
	switch browser {
	case "", "default":
		return "Open in Browser"
	case "chrome":
		return "Open in Browser(Chrome)"
	case "firefox":
		return "Open in Browser(Firefox)"
	case "opera":
		return "Open in Browser(Opera)"
	default:
		return "Open in Browser"
	}
}