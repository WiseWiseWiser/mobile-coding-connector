package screen

// lockProbe reads the host's screen-lock state. It is a variable so tests can
// simulate a locked host without locking the machine running them.
var lockProbe = screenLocked

// hostLocked reports whether the login window currently owns the display.
func hostLocked() bool {
	return lockProbe()
}

// inputInfo describes the injection backend including the current lock state.
func (m *Manager) inputInfo() InputInfo {
	info := m.backend.info()
	info.Locked = hostLocked()
	return info
}
