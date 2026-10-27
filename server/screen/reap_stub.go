//go:build !darwin

package screen

// reapStaleCapture is a no-op outside macOS: the X11 capture path does not
// claim the display device exclusively.
func reapStaleCapture() {}
