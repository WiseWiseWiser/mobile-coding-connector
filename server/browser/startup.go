package browser

import (
	"context"
	"errors"
	"fmt"
)

var errBrowserOffline = errors.New("no connected Chrome window; start or reconnect Chrome")
var errBrowserStarting = errors.New("Chrome is already starting; wait for the current request")

// StartBrowser is an explicit user action. Never let multiple viewers turn one
// stalled extension handshake into several Chrome windows or session pages.
func (m *Manager) StartBrowser(ctx context.Context) error {
	if !m.startMu.TryLock() {
		return errBrowserStarting
	}
	defer m.startMu.Unlock()
	m.setStartup("starting")
	m.ensureMu.Lock()
	// Start/Reconnect reuses the normal profile, not a stale pending +New request.
	m.mu.Lock()
	m.newWindowFor = ""
	m.mu.Unlock()
	err := m.ensureLocked(ctx, true)
	m.ensureMu.Unlock()
	if err != nil {
		m.setStartup("failed")
		return fmt.Errorf("start or reconnect Chrome: %w", err)
	}
	m.setStartup("ready")
	if m.hub != nil {
		m.hub.kick()
	}
	return nil
}

func (m *Manager) setStartup(stage string) {
	m.mu.Lock()
	m.startup = stage
	m.mu.Unlock()
	m.broadcastStatus()
}

func (m *Manager) setStartupWaiting() {
	m.mu.Lock()
	if m.startup == "starting" {
		m.startup = "waiting_extension"
	}
	m.mu.Unlock()
	m.broadcastStatus()
}

// A disconnected extension's last tab/window IDs are not live evidence. Do
// not delete a daemon session or close Chrome: clear only our ephemeral cache.
func (m *Manager) markDisconnected(sessionID string) {
	m.mu.Lock()
	if m.sessionIDValueLocked() != sessionID {
		m.mu.Unlock()
		return
	}
	changed := m.connected || m.windowID != 0 || m.contentTabID != 0 || len(m.tabs) != 0
	m.connected = false
	m.windowID, m.contentTabID, m.activeTabID = 0, 0, 0
	m.tabs, m.url = nil, ""
	m.viewport = Viewport{}
	m.presetApplied, m.macKnown = false, false
	m.mu.Unlock()
	if changed {
		m.broadcastStatus()
	}
}
