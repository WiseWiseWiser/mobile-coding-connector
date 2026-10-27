//go:build !darwin || !cgo

package screen

// screenLocked always reports false where the host lock state is not exposed:
// Linux lock screens are per-session and X11 provides no reliable signal, so
// input keeps working there (the window manager still decides what receives it).
func screenLocked() bool { return false }
