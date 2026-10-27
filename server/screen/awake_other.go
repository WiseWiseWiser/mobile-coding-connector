//go:build !darwin

package screen

// startAwakeLocked is a no-op off macOS: the capture backends there do not hand
// back a stale frame when the host display sleeps.
func (m *Manager) startAwakeLocked() {}

func (m *Manager) stopAwakeLocked() {}
