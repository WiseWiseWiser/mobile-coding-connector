package wsproxy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cloudflareSettings "github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
	"github.com/xhd2015/ai-critic/server/subprocess"
)

// Each instance owns its own xray process, config file and tunnel mapping.
// That keeps start/stop of one instance from touching the others and lets the
// legacy single-instance machinery retire without a risky shared-state
// rewrite. Instance counts here are small (1-5) so the per-instance process
// cost is negligible.

func xrayProcIDFor(id string) string    { return xrayProcID + "-" + id }
func cfQuickProcIDFor(id string) string { return cfQuickProcID + "-" + id }
func mappingIDFor(id string) string     { return mappingID + "-" + id }

func xrayConfigPathFor(id string) string {
	if _testConfigDir != "" {
		return filepath.Join(_testConfigDir, "ws-xray-"+id+".json")
	}
	return filepath.Join(xrayDir(), "ws-xray-"+id+".json")
}

// instanceRuntime is the derived per-instance runtime view.
type instanceRuntime struct {
	inst     *Instance
	cfg      *Config
	procID   string
	mappingD string
	hostname string
	port     int
}

func (m *Manager) runtimeFor(inst *Instance) (*instanceRuntime, error) {
	cfg := inst.toConfig()
	domain, err := resolveDomain()
	if err != nil {
		return nil, err
	}
	port := inst.ListenPort
	if port <= 0 {
		port = pickFreePort()
	}
	return &instanceRuntime{
		inst:     inst,
		cfg:      cfg,
		procID:   xrayProcIDFor(inst.ID),
		mappingD: mappingIDFor(inst.ID),
		hostname: inst.HostnameFor(domain),
		port:     port,
	}, nil
}

// generateXrayConfigFor writes the per-instance xray config: the outbound
// follows the backend type — freedom for self (host is the proxy), http for
// a chained upstream.
func generateXrayConfigFor(rt *instanceRuntime) error {
	var outbound xrayOutbound
	if rt.inst.BackendType == BackendProxy {
		outbound = xrayOutbound{
			Tag:      xrayOutboundProxy,
			Protocol: "http",
			Settings: map[string]any{
				"servers": []map[string]any{{
					"address": extractHost(rt.inst.BackendURL),
					"port":    extractPort(rt.inst.BackendURL),
				}},
			},
		}
	} else {
		outbound = xrayOutbound{
			Tag:      xrayOutboundDirect,
			Protocol: "freedom",
			Settings: map[string]any{},
		}
	}

	xrayCfg := xrayConfig{
		Inbounds: []xrayInbound{
			{
				Port:     rt.port,
				Protocol: "vmess",
				Settings: xrayInboundSettings{
					Clients: []xrayClient{{ID: rt.inst.UUID, AlterID: 0}},
				},
				StreamSettings: xrayStreamSettings{
					Network: "ws",
					WSSettings: xrayWSSettings{
						Path: rt.inst.WSPath,
					},
				},
				Sniffing: vmessSniffing(),
			},
		},
		Outbounds: []xrayOutbound{outbound},
	}
	withDNSHandling(&xrayCfg)

	if err := os.MkdirAll(xrayDir(), 0755); err != nil {
		return fmt.Errorf("failed to create xray dir: %w", err)
	}
	data, err := json.MarshalIndent(xrayCfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(xrayConfigPathFor(rt.inst.ID), data, 0644)
}

// startXrayFor launches the instance's xray process; an orphan from a previous
// server version that already serves the port is adopted instead of spawned.
func (m *Manager) startXrayFor(rt *instanceRuntime) error {
	err := m.startXrayOnceFor(rt)
	if err == nil {
		return nil
	}
	// A binary downloaded for another platform (or left by an older build)
	// exec-fails forever; drop it once and re-download for this platform.
	if strings.Contains(err.Error(), "exec format error") && os.Remove(xrayBinaryPath()) == nil {
		fmt.Printf("[ws-proxy] removed xray binary for wrong platform; re-downloading\n")
		return m.startXrayOnceFor(rt)
	}
	return err
}

func (m *Manager) startXrayOnceFor(rt *instanceRuntime) error {
	mgr := subprocess.GetManager()
	if mgr.IsRunning(rt.procID) {
		return nil
	}
	if isXrayAlive(rt.port, rt.inst.WSPath) {
		return nil // adopt orphan serving this exact port/path
	}
	if err := ensureXrayBinary(); err != nil {
		return fmt.Errorf("failed to setup xray: %w", err)
	}
	if err := generateXrayConfigFor(rt); err != nil {
		return fmt.Errorf("failed to generate xray config: %w", err)
	}
	cmd := exec.Command(xrayBinaryPath(), "run", "-c", xrayConfigPathFor(rt.inst.ID))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	proc, err := mgr.StartProcess(rt.procID, "ws-proxy-xray-"+rt.inst.ID, cmd, func() bool {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", rt.port), 2*time.Second)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	})
	if err != nil {
		return fmt.Errorf("failed to start xray: %w", err)
	}
	if !proc.WaitForRunning(30 * time.Second) {
		mgr.StopProcess(rt.procID)
		return fmt.Errorf("xray failed to become healthy within 30s")
	}
	return nil
}

// addInstanceTunnelMapping publishes hostname → localhost:port on the
// extension tunnel group (multi-hostname capable) with a per-instance mapping.
// The group is auto-configured when missing (find-or-create tunnel), so a
// fresh server does not require manual tunnel setup.
func (m *Manager) addInstanceTunnelMapping(rt *instanceRuntime, logFn func(string)) error {
	log := func(msg string) {
		if logFn != nil {
			logFn(msg)
		}
	}
	reconcileExtensionTunnelConnectors("ws-proxy instance")

	if _, _, _, err := cloudflareSettings.EnsureGroupTunnelConfigured(unified_tunnel.GroupExtension, "", logFn); err != nil {
		return fmt.Errorf("ensure extension tunnel: %w", err)
	}

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

	log("DNS route: " + rt.hostname)
	if err := unified_tunnel.CreateDNSRoute(tunnelRef, rt.hostname); err != nil {
		log(fmt.Sprintf("warning: DNS route error for %s: %v", rt.hostname, err))
	}
	log(fmt.Sprintf("ingress: %s → localhost:%d", rt.hostname, rt.port))
	if err := tg.AddMapping(&unified_tunnel.IngressMapping{
		ID:       rt.mappingD,
		Hostname: rt.hostname,
		Service:  fmt.Sprintf("http://localhost:%d", rt.port),
		Source:   "wsproxy",
	}); err != nil {
		return fmt.Errorf("failed to add tunnel mapping: %w", err)
	}
	return nil
}

// StartInstance brings one instance up: xray process + tunnel mapping.
func (m *Manager) StartInstance(id string) error {
	return m.StartInstanceWithLog(id, nil)
}

// StartInstanceWithLog runs the same startup while emitting progress lines.
// Slow steps (xray download on first run, tunnel find-or-create) are exactly
// the ones users need visibility into.
func (m *Manager) StartInstanceWithLog(id string, logFn func(string)) error {
	log := func(msg string) {
		if logFn != nil {
			logFn(msg)
		}
		fmt.Printf("[ws-proxy] %s\n", msg)
	}
	instances, err := LoadInstances()
	if err != nil {
		return err
	}
	inst, _ := FindInstance(instances, id)
	if inst == nil {
		return fmt.Errorf("instance %q not found", id)
	}
	rt, err := m.runtimeFor(inst)
	if err != nil {
		return err
	}
	log(fmt.Sprintf("instance %s: backend %s, listen :%d", inst.ID, backendLabel(inst), rt.port))

	log("ensure xray binary…")
	if err := m.startXrayFor(rt); err != nil {
		return err
	}
	log(fmt.Sprintf("xray up on :%d", rt.port))

	if err := m.addInstanceTunnelMapping(rt, log); err != nil {
		return err
	}

	inst.ListenPort = rt.port
	inst.PublicURL = "https://" + rt.hostname
	log("published: " + inst.PublicURL)
	return SaveInstances(instances)
}

// StopInstance tears one instance down; its xray process and mapping are
// removed while other instances keep running.
func (m *Manager) StopInstance(id string) error {
	instances, err := LoadInstances()
	if err != nil {
		return err
	}
	inst, _ := FindInstance(instances, id)
	if inst == nil {
		return fmt.Errorf("instance %q not found", id)
	}
	mgr := subprocess.GetManager()
	mgr.StopProcess(xrayProcIDFor(id))
	tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
	if tg != nil {
		_ = tg.RemoveMapping(mappingIDFor(id))
	}
	inst.PublicURL = ""
	return SaveInstances(instances)
}

// StartAllEnabled starts every enabled instance; it fails only when none
// comes up, and the returned error names the per-instance causes.
func (m *Manager) StartAllEnabled() error {
	instances, err := LoadInstances()
	if err != nil {
		return err
	}
	var failures []string
	started := 0
	for _, inst := range instances {
		if !inst.Enabled {
			continue
		}
		if err := m.StartInstance(inst.ID); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", inst.ID, err))
			continue
		}
		started++
	}
	if started == 0 && len(failures) > 0 {
		return fmt.Errorf("no instance started: %s", strings.Join(failures, "; "))
	}
	for _, f := range failures {
		fmt.Printf("[ws-proxy] instance start failure: %s\n", f)
	}
	return nil
}

// StopAllInstances stops every instance.
func (m *Manager) StopAllInstances() {
	instances, err := LoadInstances()
	if err != nil {
		return
	}
	for _, inst := range instances {
		_ = m.StopInstance(inst.ID)
	}
}

// instanceRunning reports whether the instance serves clients: its xray is
// alive AND the cloudflare ingress mapping for its hostname exists (without
// the mapping clients get ERR_PROXY_CONNECTION_FAILED, so process-liveness
// alone must not count as running).
func (m *Manager) instanceRunning(inst *Instance) bool {
	if inst.PublicURL == "" || inst.ListenPort <= 0 {
		return false
	}
	if !subprocess.GetManager().IsRunning(xrayProcIDFor(inst.ID)) && !isXrayAlive(inst.ListenPort, inst.WSPath) {
		return false
	}
	rt := &instanceRuntime{
		inst:     inst,
		mappingD: mappingIDFor(inst.ID),
		hostname: HostFromPublicURL(inst.PublicURL),
		port:     inst.ListenPort,
	}
	return hasInstanceTunnelMapping(rt)
}

// InstanceWithStatus is an instance plus its live state for the API/UI.
type InstanceWithStatus struct {
	Instance
	Running       bool   `json:"running"`
	EffectivePort int    `json:"effective_port,omitempty"`
	Backend       string `json:"backend"` // human label: "direct" | the upstream URL
}

// ListInstancesWithStatus renders the store with live flags.
func (m *Manager) ListInstancesWithStatus() []InstanceWithStatus {
	instances, err := LoadInstances()
	if err != nil {
		return nil
	}
	out := make([]InstanceWithStatus, 0, len(instances))
	for _, inst := range instances {
		ws := InstanceWithStatus{Instance: inst, Backend: backendLabel(&inst)}
		if m.instanceRunning(&inst) {
			ws.Running = true
			ws.EffectivePort = inst.ListenPort
		}
		out = append(out, ws)
	}
	return out
}

func backendLabel(inst *Instance) string {
	if inst.BackendType == BackendProxy {
		return "HTTP proxy · " + inst.BackendURL
	}
	return "This server · direct"
}

// InstanceVMessLink returns the scan-able link for a running instance.
func (m *Manager) InstanceVMessLink(id string) (string, *VMessConfig, error) {
	instances, err := LoadInstances()
	if err != nil {
		return "", nil, err
	}
	inst, _ := FindInstance(instances, id)
	if inst == nil {
		return "", nil, fmt.Errorf("instance %q not found", id)
	}
	if !m.instanceRunning(inst) {
		return "", nil, fmt.Errorf("instance %q is not running", id)
	}
	host := strings.TrimPrefix(inst.PublicURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	vmess := map[string]interface{}{
		"v":    "2",
		"ps":   "ws-proxy-" + inst.Name,
		"add":  host,
		"port": "443",
		"id":   inst.UUID,
		"aid":  "0",
		"net":  "ws",
		"type": "none",
		"host": host,
		"path": inst.WSPath,
		"tls":  "tls",
		"sni":  host,
		"scy":  "auto",
	}
	data, _ := json.Marshal(vmess)
	link := "vmess://" + base64.StdEncoding.EncodeToString(data)
	cfg := &VMessConfig{
		Host:    host,
		Port:    "443",
		UUID:    inst.UUID,
		AlterID: "0",
		Network: "ws",
		Type:    "none",
		Path:    inst.WSPath,
		TLS:     "tls",
	}
	return link, cfg, nil
}

// InstanceReachable is a cheap per-instance aliveness check for tests.
func InstanceReachable(port int, wsPath string) bool {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, wsPath))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 400
}
