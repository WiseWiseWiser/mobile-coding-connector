package wsproxy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
	"github.com/xhd2015/ai-critic/server/subprocess"
)

func (m *Manager) hydrateFromConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if m.publicURL == "" && cfg.PublicURL != "" {
		m.publicURL = cfg.PublicURL
		m.isTmp = cfg.IsTmp
	}
}

func (m *Manager) effectivePublicURL(cfg *Config) string {
	if m.publicURL != "" {
		return m.publicURL
	}
	if cfg != nil && cfg.PublicURL != "" {
		return cfg.PublicURL
	}
	if cfg != nil && !cfg.IsTmp {
		domain, err := resolveDomain()
		if err == nil {
			return DerivePublicURL(cfg, domain)
		}
	}
	return ""
}

func (m *Manager) isLocalXrayAlive(cfg *Config, port int) bool {
	if cfg == nil {
		return false
	}
	if subprocess.GetManager().IsRunning(xrayProcID) {
		return true
	}
	return isXrayAlive(port, cfg.WSPath)
}

func (m *Manager) isTunnelReady(cfg *Config, publicURL string, isTmp bool, port int) bool {
	if publicURL == "" {
		return false
	}
	if isTmp {
		return subprocess.GetManager().IsRunning(cfQuickProcID)
	}
	return hasTunnelMapping(HostFromPublicURL(publicURL), port)
}

func (m *Manager) isClientReady(cfg *Config, publicURL string, port int) bool {
	return m.isLocalXrayAlive(cfg, port) &&
		m.isTunnelReady(cfg, publicURL, m.isTmp, port) &&
		publicURL != "" &&
		cfg != nil && cfg.UUID != ""
}

var _testTunnelMapped map[string]bool

// SetTestTunnelMapped overrides tunnel ingress checks for unit tests.
func SetTestTunnelMapped(hostname string, listenPort int, mapped bool) {
	if _testTunnelMapped == nil {
		_testTunnelMapped = make(map[string]bool)
	}
	_testTunnelMapped[fmt.Sprintf("%s:%d", hostname, listenPort)] = mapped
}

func clearTestTunnelMapped() {
	_testTunnelMapped = nil
}

func hasTunnelMapping(hostname string, listenPort int) bool {
	if hostname == "" || listenPort <= 0 {
		return false
	}
	if _testConfigDir != "" && _testTunnelMapped != nil {
		if mapped, ok := _testTunnelMapped[fmt.Sprintf("%s:%d", hostname, listenPort)]; ok {
			return mapped
		}
	}
	tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
	mapping, ok := tg.GetMapping(mappingID)
	if !ok {
		return false
	}
	if !strings.EqualFold(mapping.Hostname, hostname) {
		return false
	}
	expected := fmt.Sprintf("http://localhost:%d", listenPort)
	return mapping.Service == expected
}

func (m *Manager) persistRuntimeState(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	cfg.PublicURL = m.publicURL
	cfg.IsTmp = m.isTmp
	return SaveConfig(cfg)
}

func (m *Manager) clearPersistedRuntimeState(cfg *Config) {
	if cfg == nil {
		return
	}
	cfg.PublicURL = ""
	cfg.IsTmp = false
	_ = SaveConfig(cfg)
}

// Recover re-adds missing Cloudflare ingress mappings when local xray is
// still healthy. Instance-aware: every running instance gets its mapping
// reconciled; the legacy single-instance recover stays as fallback.
func (m *Manager) Recover() error {
	instances, err := LoadInstances()
	if err == nil {
		for _, inst := range instances {
			if inst.PublicURL == "" || inst.ListenPort <= 0 {
				continue
			}
			if !subprocess.GetManager().IsRunning(xrayProcIDFor(inst.ID)) && !isXrayAlive(inst.ListenPort, inst.WSPath) {
				continue
			}
			hostname := HostFromPublicURL(inst.PublicURL)
			rt := &instanceRuntime{
				inst:     &inst,
				mappingD: mappingIDFor(inst.ID),
				hostname: hostname,
			}
			if hasInstanceTunnelMapping(rt) {
				continue
			}
			// No extension tunnel configured (yet): nothing to reconcile.
			if unified_tunnel.GetTunnelGroupManager().GetExtensionGroup().GetConfig() == nil {
				continue
			}
			fmt.Printf("[ws-proxy] Recover: re-adding mapping for %s → localhost:%d\n", hostname, inst.ListenPort)
			if err := m.addInstanceTunnelMapping(rt, nil); err != nil {
				fmt.Printf("[ws-proxy] Recover failed for %s: %v\n", inst.ID, err)
			}
		}
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.recoverLocked()
}

// hasInstanceTunnelMapping checks whether the instance's hostname mapping is
// already present on the extension tunnel group.
func hasInstanceTunnelMapping(rt *instanceRuntime) bool {
	if _testConfigDir != "" && _testTunnelMapped != nil {
		if mapped, ok := _testTunnelMapped[fmt.Sprintf("%s:%d", rt.hostname, rt.port)]; ok {
			return mapped
		}
	}
	tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
	mapping, ok := tg.GetMapping(rt.mappingD)
	if !ok {
		return false
	}
	if !strings.EqualFold(mapping.Hostname, rt.hostname) {
		return false
	}
	return mapping.Service == fmt.Sprintf("http://localhost:%d", rt.port)
}

func reconcileExtensionTunnelConnectors(context string) {
	tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
	killed, err := tg.TunnelMgr().ReconcileStaleConnectors()
	if err != nil {
		fmt.Printf("[ws-proxy] %s: stale connector cleanup error: %v\n", context, err)
		return
	}
	if len(killed) > 0 {
		fmt.Printf("[ws-proxy] %s: killed stale cloudflared connector PIDs: %s\n",
			context, strings.Join(intSliceToString(killed), ", "))
	}
}

func intSliceToString(nums []int) []string {
	out := make([]string, len(nums))
	for i, n := range nums {
		out[i] = strconv.Itoa(n)
	}
	return out
}

func (m *Manager) recoverLocked() error {
	reconcileExtensionTunnelConnectors("Recover")

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	m.hydrateFromConfig(cfg)

	if cfg.UpstreamProxy == "" {
		return nil
	}

	port := resolvePort(cfg)
	if !m.isLocalXrayAlive(cfg, port) {
		return nil
	}

	publicURL := m.effectivePublicURL(cfg)
	if publicURL == "" {
		return nil
	}

	if m.isTunnelReady(cfg, publicURL, m.isTmp, port) {
		return nil
	}

	if m.isTmp {
		fmt.Printf("[ws-proxy] Recover: quick tunnel is down; run ws-proxy start --tmp to recreate\n")
		return nil
	}

	hostname := HostFromPublicURL(publicURL)
	fmt.Printf("[ws-proxy] Recover: re-adding tunnel mapping for %s → localhost:%d\n", hostname, port)
	return m.addPermanentTunnelMappingLocked(cfg, hostname, port)
}

func (m *Manager) addPermanentTunnelMapping(cfg *Config, hostname string, listenPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addPermanentTunnelMappingLocked(cfg, hostname, listenPort)
}

func (m *Manager) addPermanentTunnelMappingLocked(cfg *Config, hostname string, listenPort int) error {
	reconcileExtensionTunnelConnectors("start")

	localURL := fmt.Sprintf("http://localhost:%d", listenPort)

	tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
	tgCfg := tg.GetConfig()
	if tgCfg == nil {
		return fmt.Errorf("extension tunnel is not configured")
	}

	tunnelRef := tgCfg.TunnelName
	if tunnelRef == "" {
		tunnelRef = tgCfg.TunnelID
	}
	if tunnelRef == "" {
		return fmt.Errorf("extension tunnel has no tunnel name or ID")
	}

	if err := unified_tunnel.CreateDNSRoute(tunnelRef, hostname); err != nil {
		fmt.Printf("[ws-proxy] Warning: DNS route error: %v\n", err)
	}

	mapping := &unified_tunnel.IngressMapping{
		ID:       mappingID,
		Hostname: hostname,
		Service:  localURL,
		Source:   "wsproxy",
	}
	if err := tg.AddMapping(mapping); err != nil {
		return fmt.Errorf("failed to add tunnel mapping: %w", err)
	}

	publicURL := fmt.Sprintf("https://%s", hostname)
	m.publicURL = publicURL
	m.isTmp = false
	return m.persistRuntimeState(cfg)
}
