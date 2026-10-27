//go:build darwin

package screen

import (
	"log"
	"os/exec"
)

// caffeinateCommand is a seam so tests can watch the lifecycle without holding
// a real power assertion.
var caffeinateCommand = func() *exec.Cmd {
	// -d: keep the display on, -i: no idle system sleep, -u: declare user
	// activity so a display that already slept wakes up again.
	return exec.Command("caffeinate", "-d", "-i", "-u")
}

// startAwakeLocked holds a power assertion for as long as somebody is watching.
// Without it macOS sleeps the display as soon as the host user walks away and
// AVFoundation keeps returning the frame it captured before sleeping, so the
// mirror freezes on whatever was on screen at that moment — exactly when a
// remote viewer needs it most.
func (m *Manager) startAwakeLocked() {
	if m.awake != nil {
		return
	}
	cmd := caffeinateCommand()
	if err := cmd.Start(); err != nil {
		log.Printf("[screen] cannot hold display awake: %v", err)
		return
	}
	m.awake = cmd
	log.Printf("[screen] holding display awake while viewers are connected")
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		if m.awake == cmd {
			m.awake = nil
		}
		m.mu.Unlock()
	}()
}

// stopAwakeLocked drops the assertion once the last viewer is gone, so a Mac
// that nobody watches sleeps normally.
func (m *Manager) stopAwakeLocked() {
	if m.awake == nil {
		return
	}
	if err := m.awake.Process.Kill(); err != nil {
		log.Printf("[screen] release display assertion: %v", err)
	}
	m.awake = nil
	log.Printf("[screen] released display sleep assertion")
}
