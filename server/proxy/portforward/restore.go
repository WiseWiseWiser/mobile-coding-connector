package portforward

import (
	"fmt"
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
	t, ok := m.tunnels[config.LocalMappingID(port)]
	if !ok {
		return "", false, false
	}
	if t.hopAlive != nil && !t.hopAlive() {
		return t.status, true, true
	}
	return t.status, true, false
}

func (m *Manager) restoreState(port string) *restoreAttempt {
	m.restoreMu.Lock()
	defer m.restoreMu.Unlock()
	if m.restore == nil {
		m.restore = map[string]*restoreAttempt{}
	}
	st := m.restore[port]
	if st == nil {
		st = &restoreAttempt{}
		m.restore[port] = st
	}
	return st
}

func (m *Manager) dropRestoreState(keep map[string]struct{}) {
	m.restoreMu.Lock()
	defer m.restoreMu.Unlock()
	for port := range m.restore {
		if _, ok := keep[port]; !ok {
			delete(m.restore, port)
		}
	}
}

func (m *Manager) markRestoreFail(port string) {
	st := m.restoreState(port)
	st.failures++
	st.stableSince = time.Time{}
	st.nextRetry = m.nowTime().Add(restoreBackoffDelay(st.failures))
}

func (m *Manager) markRestoreOK(port string) {
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
// ensureDesired remains the legacy port entry point used by services/tests.
func (m *Manager) ensureDesired(port int, spec config.PortMappingDesiredSpec) error {
	var auth config.PortAuth
	if saved, err := config.GetPortMappingAuth(port); err != nil {
		return err
	} else if saved != nil {
		auth = *saved
	}
	return m.ensureRecord(config.MappingRecord{ID: config.LocalMappingID(port), Target: config.OriginTarget{Host: "127.0.0.1", Scheme: "http", Port: port}, Hostname: spec.Hostname, Provider: spec.Provider, Auth: auth, Desired: true})
}
func (m *Manager) ensureRecord(rec config.MappingRecord) error {
	m.mu.Lock()
	t, ok := m.tunnels[rec.ID]
	active := ok && t.status == StatusActive && (t.hopAlive == nil || t.hopAlive())
	m.mu.Unlock()
	if active {
		m.markRestoreOK(rec.ID)
		return nil
	}
	if ok {
		_ = m.RemoveSelector(rec.ID)
	}
	if strings.TrimSpace(rec.Hostname) == "" {
		return fmt.Errorf("desired mapping %s has no hostname", rec.ID)
	}
	if rec.Provider == "" {
		rec.Provider = ProviderCloudflareOwned
	}
	pf, err := m.AddForward(recordForward(rec))
	if err != nil {
		m.markRestoreFail(rec.ID)
		return err
	}
	if pf != nil && pf.Status == StatusError {
		m.markRestoreFail(rec.ID)
		return fmt.Errorf("%s", pf.Error)
	}
	m.markRestoreOK(rec.ID)
	return nil
}
func (m *Manager) RestoreDesired() { m.restoreTick() }
func (m *Manager) restoreTick() {
	mappingMutationMu.Lock()
	defer mappingMutationMu.Unlock()
	reconcileOwnedConnectors()
	store, err := config.LoadMappings()
	if err != nil {
		fmt.Printf("[port mapping] load mappings: %v\n", err)
		return
	}
	keep := map[string]struct{}{}
	now := m.nowTime()
	for id, rec := range store.Mappings {
		if !rec.Desired {
			continue
		}
		keep[id] = struct{}{}
		st := m.restoreState(id)
		if !st.nextRetry.IsZero() && now.Before(st.nextRetry) {
			continue
		}
		if err := m.ensureRecord(rec); err != nil {
			fmt.Printf("[port mapping] restore %s: %v\n", id, err)
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
	StartConnectorWatchdog()
}
