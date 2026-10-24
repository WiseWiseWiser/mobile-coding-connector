package portforward

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

const (
	restoreTick       = 5 * time.Second
	restoreBackoffMin = 2 * time.Second
	restoreBackoffCap = 30 * time.Second
	restoreStableFor  = 30 * time.Second
)

type restoreAttempt struct {
	failures    int
	nextRetry   time.Time
	stableSince time.Time
}

func restoreBackoffDelay(failures int) time.Duration {
	if failures <= 0 {
		return 0
	}
	delay := restoreBackoffMin
	for i := 1; i < failures; i++ {
		if delay >= restoreBackoffCap/2 {
			return restoreBackoffCap
		}
		delay *= 2
	}
	if delay > restoreBackoffCap {
		return restoreBackoffCap
	}
	return delay
}

func (m *Manager) nowTime() time.Time {
	if m != nil && m.now != nil {
		return m.now()
	}
	return time.Now()
}

func (m *Manager) forwardStatus(port int) (string, bool) {
	status, ok, _ := m.forwardHealth(port)
	return status, ok
}

func (m *Manager) forwardHealth(port int) (status string, ok bool, hopDead bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tunnels[port]
	if !ok {
		return "", false, false
	}
	if t.hopAlive != nil && !t.hopAlive() {
		return t.status, true, true
	}
	return t.status, true, false
}

func (m *Manager) restoreState(port int) *restoreAttempt {
	m.restoreMu.Lock()
	defer m.restoreMu.Unlock()
	if m.restore == nil {
		m.restore = map[int]*restoreAttempt{}
	}
	st := m.restore[port]
	if st == nil {
		st = &restoreAttempt{}
		m.restore[port] = st
	}
	return st
}

func (m *Manager) dropRestoreState(keep map[int]struct{}) {
	m.restoreMu.Lock()
	defer m.restoreMu.Unlock()
	for port := range m.restore {
		if _, ok := keep[port]; !ok {
			delete(m.restore, port)
		}
	}
}

func (m *Manager) markRestoreFail(port int) {
	st := m.restoreState(port)
	st.failures++
	st.stableSince = time.Time{}
	st.nextRetry = m.nowTime().Add(restoreBackoffDelay(st.failures))
}

func (m *Manager) markRestoreOK(port int) {
	st := m.restoreState(port)
	now := m.nowTime()
	if st.stableSince.IsZero() {
		st.stableSince = now
	}
	if now.Sub(st.stableSince) >= restoreStableFor {
		st.failures = 0
	}
	st.nextRetry = time.Time{}
}

// ensureDesired publishes one desired mapping. An already-active forward is
// left alone (including a service that owns the port). Error/missing entries
// are rebuilt, reusing saved auth tokens.
func (m *Manager) ensureDesired(port int, spec config.PortMappingDesiredSpec) error {
	hostname := strings.TrimSpace(spec.Hostname)
	if hostname == "" {
		return fmt.Errorf("desired mapping for port %d has no hostname", port)
	}
	status, ok, hopDead := m.forwardHealth(port)
	if ok && status == StatusActive && !hopDead {
		m.markRestoreOK(port)
		return nil
	}
	if ok {
		if err := m.Remove(port); err != nil && !strings.Contains(err.Error(), "not being forwarded") {
			return err
		}
	}

	var auth config.PortAuth
	if saved, err := config.GetPortMappingAuth(port); err != nil {
		return err
	} else if saved != nil && saved.Required {
		resolved, err := resolveForwardAuth(port, saved)
		if err != nil {
			return err
		}
		auth = resolved
	}
	provider := strings.TrimSpace(spec.Provider)
	if provider == "" {
		provider = ProviderCloudflareOwned
	}
	pf, err := m.AddForward(ForwardRequest{
		Port:     port,
		Label:    hostname,
		Provider: provider,
		Auth:     auth,
	})
	if err != nil {
		if strings.Contains(err.Error(), "already being forwarded") {
			fmt.Printf("warning: port %d desired mapping skipped: already forwarded\n", port)
			m.markRestoreOK(port)
			return nil
		}
		m.markRestoreFail(port)
		return err
	}
	if pf != nil && pf.Status == StatusError {
		m.markRestoreFail(port)
		errMsg := pf.Error
		if errMsg == "" {
			errMsg = "forward status error"
		}
		return fmt.Errorf("%s", errMsg)
	}
	m.markRestoreOK(port)
	return nil
}

// RestoreDesired publishes every desired-live mapping once (boot).
func (m *Manager) RestoreDesired() {
	reconcileOwnedConnectors()
	desired, err := config.LoadPortMappingDesired()
	if err != nil {
		fmt.Printf("[port mapping] load desired: %v\n", err)
		return
	}
	for portStr, spec := range desired {
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 {
			continue
		}
		if err := m.ensureDesired(port, spec); err != nil {
			fmt.Printf("[port mapping] restore port %d: %v\n", port, err)
		}
	}
}

func (m *Manager) restoreTick() {
	reconcileOwnedConnectors()
	desired, err := config.LoadPortMappingDesired()
	if err != nil {
		fmt.Printf("[port mapping] watchdog load desired: %v\n", err)
		return
	}
	keep := map[int]struct{}{}
	now := m.nowTime()
	for portStr, spec := range desired {
		port, convErr := strconv.Atoi(portStr)
		if convErr != nil || port <= 0 {
			continue
		}
		keep[port] = struct{}{}
		st := m.restoreState(port)
		if !st.nextRetry.IsZero() && now.Before(st.nextRetry) {
			continue
		}
		if err := m.ensureDesired(port, spec); err != nil {
			fmt.Printf("[port mapping] watchdog port %d: %v (retry in %s)\n", port, err, restoreBackoffDelay(st.failures))
		}
	}
	m.dropRestoreState(keep)
}

// StartRestoreWatchdog retries desired mappings that are missing or in error.
func (m *Manager) StartRestoreWatchdog() {
	m.restoreMu.Lock()
	if m.restoreStop != nil {
		m.restoreMu.Unlock()
		return
	}
	m.restoreStop = make(chan struct{})
	stop := m.restoreStop
	m.restoreMu.Unlock()

	go func() {
		ticker := time.NewTicker(restoreTick)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				m.restoreTick()
			}
		}
	}()
}

func (m *Manager) StopRestoreWatchdog() {
	m.restoreMu.Lock()
	defer m.restoreMu.Unlock()
	if m.restoreStop != nil {
		close(m.restoreStop)
		m.restoreStop = nil
	}
}

// AutoRestoreDesiredMappings restores persisted live mappings and starts the
// watchdog. Safe to call from server extension startup.
func AutoRestoreDesiredMappings() {
	defaultManager.RestoreDesired()
	defaultManager.StartRestoreWatchdog()
}
