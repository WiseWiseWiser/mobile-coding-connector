// Package systemservices wires the server's own in-process subsystems into the
// services manager, so the GUI and the CLI can list and control them next to the
// user's own services ("System Services").
//
// A system service has no command and no services.json entry: it exists because
// the server owns the subsystem, and its lifecycle methods delegate straight to
// that subsystem's manager.
package systemservices

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xhd2015/agent-pro/agent/streaming/sse"
	"github.com/xhd2015/ai-critic/server/gomod"
	"github.com/xhd2015/ai-critic/server/hostmon"
	"github.com/xhd2015/ai-critic/server/openclaw"
	"github.com/xhd2015/ai-critic/server/proxy/httpproxy"
	"github.com/xhd2015/ai-critic/server/proxy/wsproxy"
	"github.com/xhd2015/ai-critic/server/services"
)

// Stable ids: the CLI and the GUI address system services by them. The sys-
// prefix keeps them clear of the generated svc- ids.
const (
	WSProxyID   = "sys-ws-proxy"
	HTTPProxyID = "sys-http-proxy"
	CFProxyID   = "sys-cloudflare-proxy"
	GoModID     = "sys-go-mod-proxy"
	OpenClawID  = "sys-openclaw"
	HostMonID   = "sys-host-monitor"
)

// Register adds every system service to m. Registration order is the order the
// GUI lists them. Registering twice is a no-op, so every server start path can
// call it.
func Register(m *services.Manager) error {
	registered := make(map[string]bool)
	for _, svc := range m.SystemServices() {
		registered[svc.ID] = true
	}

	entries := []services.SystemService{
		{
			ID:          WSProxyID,
			Name:        "WS Proxy",
			Description: "Mobile WebSocket proxy (Xray + Cloudflare tunnel)",
			Controller:  wsProxyController{},
		},
		{
			ID:          HTTPProxyID,
			Name:        "HTTP Proxy",
			Description: "Intranet forward proxy with mandatory Basic auth; supports CONNECT",
			Controller:  httpProxyController{},
		},
		{
			ID:          CFProxyID,
			Name:        "Cloudflare Proxy",
			Description: "Publishes this server's hostnames through the edge reverse proxy",
			Controller:  &cloudFlareProxyController{},
		},
		{
			ID:          GoModID,
			Name:        "Go Mod Proxy",
			Description: "GOPROXY server for remote agents; serves the module cache",
			Controller:  goModController{},
		},
		{
			ID:          OpenClawID,
			Name:        "OpenClaw Gateway",
			Description: "Slack socket-mode gateway; simulated until the real integration lands",
			Controller:  openClawController{},
		},
		{
			ID:          HostMonID,
			Name:        "Host Monitor",
			Description: "Samples CPU and disk I/O so a later reboot can show which process caused the load",
			Controller:  hostMonController{},
		},
	}
	for _, entry := range entries {
		if registered[entry.ID] {
			continue
		}
		if err := m.RegisterSystemService(entry); err != nil {
			return err
		}
	}
	return nil
}

// Every controller must satisfy the services contract; the auto-start half is
// asserted too because each subsystem exposes its own switch.
var (
	_ services.SystemController        = wsProxyController{}
	_ services.SystemLifecycle         = wsProxyController{}
	_ services.SystemAutoStarter       = wsProxyController{}
	_ services.SystemOperationStreamer = wsProxyController{}
	_ services.SystemController        = httpProxyController{}
	_ services.SystemLifecycle         = httpProxyController{}
	_ services.SystemAutoStarter       = httpProxyController{}
	_ services.SystemOperationStreamer = httpProxyController{}
	_ services.SystemController        = goModController{}
	_ services.SystemLifecycle         = goModController{}
	_ services.SystemAutoStarter       = goModController{}
	_ services.SystemController        = openClawController{}
	_ services.SystemLifecycle         = openClawController{}
	_ services.SystemAutoStarter       = openClawController{}
	_ services.SystemController        = hostMonController{}
	_ services.SystemLifecycle         = hostMonController{}
	_ services.SystemAutoStarter       = hostMonController{}
	_ services.SystemController        = &cloudFlareProxyController{}
	_ services.SystemRestarter         = &cloudFlareProxyController{}
	_ services.SystemHostProvider      = &cloudFlareProxyController{}
)

func boolPtr(v bool) *bool { return &v }

// wsProxyController wraps the in-process Xray/tunnel proxy.
type wsProxyController struct{}

func (wsProxyController) SystemStatus() services.SystemRuntime {
	runtime := services.SystemRuntime{}

	// Multi-instance aggregate: running/enabled counts and the domain list.
	if instances, err := wsproxy.LoadInstances(); err == nil && len(instances) > 0 {
		live := 0
		enabled := 0
		var urls []string
		for _, st := range wsproxy.GetManager().ListInstancesWithStatus() {
			if st.Running {
				live++
				urls = append(urls, st.PublicURL)
			}
			if st.Enabled {
				enabled++
			}
		}
		runtime.Detail = fmt.Sprintf("%d/%d instances", live, len(instances))
		if live == 0 {
			runtime.Detail += " · not running"
		}
		runtime.PublicURL = strings.Join(urls, ", ")
		runtime.AutoStart = boolPtr(enabled > 0)
		return runtime
	}

	// Legacy fallback (no instances store yet).
	configured := wsproxy.ConfigExists()
	if configured {
		if cfg, err := wsproxy.LoadConfig(); err == nil {
			runtime.AutoStart = boolPtr(cfg.AutoStart)
		}
	}

	status := wsproxy.GetManager().Status()
	if status == nil {
		return runtime
	}
	runtime.Running = status.Running
	runtime.Port = status.Port
	if configured || status.Running {
		runtime.PublicURL = status.PublicURL
	}

	switch {
	case status.Running:
		runtime.Detail = "listening " + formatPort(status.Port)
	case !configured:
		runtime.Detail = "not configured"
	default:
		runtime.Detail = "not running"
	}
	return runtime
}

func (wsProxyController) StartSystem() error {
	// Instance mode: start every enabled instance.
	if instances, err := wsproxy.LoadInstances(); err == nil && len(instances) > 0 {
		return wsproxy.GetManager().StartAllEnabled()
	}
	return wsproxy.GetManager().Start(false)
}

func (wsProxyController) StopSystem() error {
	return wsproxy.GetManager().Stop()
}

// StreamSystemOperation streams a start/stop across instances: each instance
// contributes its own lines, so partial failures are visible per instance.
func (wsProxyController) StreamSystemOperation(w http.ResponseWriter, action string) error {
	switch action {
	case services.SystemActionStart:
		instances, err := wsproxy.LoadInstances()
		if err != nil || len(instances) == 0 {
			return wsproxy.GetManager().StartStream(w, false)
		}
		sw := sse.NewWriter(w)
		if sw == nil {
			return fmt.Errorf("streaming unsupported")
		}
		anyOK := false
		for _, inst := range instances {
			if !inst.Enabled {
				continue
			}
			sw.SendLog("starting instance " + inst.ID + "…")
			if err := wsproxy.GetManager().StartInstance(inst.ID); err != nil {
				sw.SendError(inst.ID + ": " + err.Error())
				continue
			}
			anyOK = true
			sw.SendLog(inst.ID + ": running")
		}
		if anyOK {
			sw.SendDone(map[string]string{"status": "running", "action": action})
		} else {
			sw.SendDone(map[string]string{"status": "error", "action": action})
		}
		return nil
	case services.SystemActionStop:
		return wsProxyController{}.StreamStop(w)
	case services.SystemActionRestart:
		if err := (wsProxyController{}).StreamStop(w); err != nil {
			return err
		}
		return (wsProxyController{}).StreamSystemOperation(w, services.SystemActionStart)
	}
	return fmt.Errorf("unsupported action %q", action)
}

// StreamStop emits stop lines for every instance.
func (wsProxyController) StreamStop(w http.ResponseWriter) error {
	sw := sse.NewWriter(w)
	if sw == nil {
		return fmt.Errorf("streaming unsupported")
	}
	for _, inst := range func() []wsproxy.Instance {
		instances, err := wsproxy.LoadInstances()
		if err != nil {
			return nil
		}
		return instances
	}() {
		sw.SendLog("stopping instance " + inst.ID + "…")
		if err := wsproxy.GetManager().StopInstance(inst.ID); err != nil {
			sw.SendError(inst.ID + ": " + err.Error())
		}
	}
	sw.SendDone(map[string]string{"status": "stopped"})
	return nil
}

func (wsProxyController) SetSystemAutoStart(enabled bool) error {
	// Refuse rather than save a config built from defaults: that would freeze a
	// random instance id into the published hostname.
	if !wsproxy.ConfigExists() {
		return fmt.Errorf("ws-proxy is not configured yet; set it up before changing auto-start")
	}
	cfg, err := wsproxy.LoadConfig()
	if err != nil {
		return err
	}
	cfg.AutoStart = enabled
	return wsproxy.SaveConfig(cfg)
}

// httpProxyController wraps the in-process HTTP forward proxy.
type httpProxyController struct{}

func (httpProxyController) SystemStatus() services.SystemRuntime {
	st := httpproxy.GetManager().Status()
	runtime := services.SystemRuntime{
		Running:   st.Running,
		Port:      st.Port,
		AutoStart: &st.AutoStart,
		LogPath:   httpproxy.LogPath(),
	}
	if st.Running {
		runtime.Detail = st.Detail + " · basic auth " + st.AuthUser
		if st.PublishErr != "" {
			runtime.Detail += " · domain error"
			runtime.Status = "error"
		}
	} else {
		runtime.Detail = st.Detail
	}
	runtime.PublicURL = st.PublicURL
	return runtime
}

func (httpProxyController) StartSystem() error {
	return httpproxy.GetManager().Start()
}

func (httpProxyController) StopSystem() error {
	return httpproxy.GetManager().Stop()
}

func (httpProxyController) SetSystemAutoStart(enabled bool) error {
	cfg, err := httpproxy.LoadConfig()
	if err != nil {
		return err
	}
	cfg.AutoStart = enabled
	return httpproxy.SaveConfig(cfg)
}

// StreamSystemOperation runs an http-proxy action with live SSE lines. The
// operations are fast, so the stream emits the steps and the final status.
func (httpProxyController) StreamSystemOperation(w http.ResponseWriter, action string) error {
	sw := sse.NewWriter(w)
	if sw == nil {
		return fmt.Errorf("streaming unsupported")
	}
	m := httpproxy.GetManager()
	switch action {
	case services.SystemActionStart:
		sw.SendLog("starting http proxy…")
		if err := m.Start(); err != nil {
			// The dominant failure ("no password configured") is the same
			// message the card's state line hints at.
			sw.SendError(err.Error())
			sw.SendDone(map[string]string{"status": "error", "action": action})
			return nil
		}
		st := m.Status()
		sw.SendLog(fmt.Sprintf("listener up: %s:%d", "0.0.0.0", st.Port))
		if st.PublishErr != "" {
			sw.SendLog("domain publish failed: " + st.PublishErr)
		} else if st.Domain != "" {
			sw.SendLog("domain published: " + st.Domain)
		} else {
			sw.SendLog("no domain configured — LAN/direct only")
		}
		sw.SendLog("basic auth user: " + st.AuthUser)
		sw.SendDone(map[string]string{"status": "running", "action": action, "port": strconv.Itoa(st.Port)})
	case services.SystemActionStop:
		sw.SendLog("stopping http proxy…")
		if err := m.Stop(); err != nil {
			sw.SendError("stop failed: " + err.Error())
			sw.SendDone(map[string]string{"status": "error", "action": action})
			return nil
		}
		sw.SendLog("stopped")
		sw.SendDone(map[string]string{"status": "stopped", "action": action})
	case services.SystemActionRestart:
		sw.SendLog("restarting http proxy…")
		if err := m.Stop(); err != nil {
			sw.SendError("stop failed: " + err.Error())
			sw.SendDone(map[string]string{"status": "error", "action": action})
			return nil
		}
		if err := m.Start(); err != nil {
			sw.SendError("start failed: " + err.Error())
			sw.SendDone(map[string]string{"status": "error", "action": action})
			return nil
		}
		st := m.Status()
		sw.SendLog(fmt.Sprintf("listener up: %s:%d", "0.0.0.0", st.Port))
		sw.SendDone(map[string]string{"status": "running", "action": action, "port": strconv.Itoa(st.Port)})
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
	return nil
}

// goModController wraps the in-process GOPROXY listener.
type goModController struct{}

func (goModController) SystemStatus() services.SystemRuntime {
	manager, err := gomod.DefaultManager()
	if err != nil {
		return services.SystemRuntime{Detail: err.Error()}
	}
	kv := manager.Status()

	port, _ := strconv.Atoi(kv["port"])
	runtime := services.SystemRuntime{
		Running:   kv["status"] == "running",
		Port:      port,
		AutoStart: boolPtr(kv["auto_start"] == "enabled"),
		LogPath:   manager.LogPath(),
	}

	parts := make([]string, 0, 3)
	if runtime.Running {
		parts = append(parts, "listening "+formatPort(port))
		if uptime := strings.TrimSpace(kv["uptime"]); uptime != "" {
			parts = append(parts, "uptime "+uptime)
		}
		if modules := strings.TrimSpace(kv["modules"]); modules != "" {
			parts = append(parts, modules+" modules")
		}
	} else {
		parts = append(parts, "not running")
	}
	runtime.Detail = strings.Join(parts, " · ")
	return runtime
}

func (goModController) StartSystem() error {
	manager, err := gomod.DefaultManager()
	if err != nil {
		return err
	}
	_, err = manager.Start()
	return err
}

func (goModController) StopSystem() error {
	manager, err := gomod.DefaultManager()
	if err != nil {
		return err
	}
	_, err = manager.Stop()
	return err
}

// SetSystemAutoStart only changes boot auto-start; it never starts or stops the
// listener, matching the user-service enable/disable contract.
func (goModController) SetSystemAutoStart(enabled bool) error {
	manager, err := gomod.DefaultManager()
	if err != nil {
		return err
	}
	if enabled {
		_, err = manager.Enable(false)
		return err
	}
	_, err = manager.Disable()
	return err
}

// openClawController wraps the OpenClaw gateway integration.
type openClawController struct{}

func (openClawController) SystemStatus() services.SystemRuntime {
	runtime := services.SystemRuntime{}
	slackEnabled := false
	if cfg, err := openclaw.LoadConfig(); err == nil {
		runtime.AutoStart = boolPtr(cfg.AutoStart)
		slackEnabled = cfg.Slack != nil && cfg.Slack.Enabled
	}
	status := openclaw.GetManager().Status()
	if status == nil {
		return runtime
	}
	runtime.Running = status.Running
	runtime.Port = status.GatewayPort
	runtime.Mocked = status.Mocked
	if slackEnabled {
		runtime.Detail = "slack on"
	} else {
		runtime.Detail = "slack off"
	}
	return runtime
}

func (openClawController) StartSystem() error {
	return openclaw.GetManager().Start()
}

func (openClawController) StopSystem() error {
	return openclaw.GetManager().Stop()
}

func (openClawController) SetSystemAutoStart(enabled bool) error {
	cfg, err := openclaw.LoadConfig()
	if err != nil {
		return err
	}
	merged := openclaw.MergeConfig(cfg, openclaw.ConfigUpdate{AutoStart: boolPtr(enabled)})
	return openclaw.SaveConfig(merged)
}

// hostMonController wraps the in-process /proc sampler.
type hostMonController struct{}

func (hostMonController) SystemStatus() services.SystemRuntime {
	status := hostmon.Default().Status()
	detail := "not running"
	if status.Running {
		detail = "sampling every 15s"
	}
	return services.SystemRuntime{
		Running:   status.Running,
		AutoStart: boolPtr(status.AutoStart),
		LogPath:   status.LogPath,
		Detail:    detail,
	}
}

func (hostMonController) StartSystem() error {
	return hostmon.Default().Start()
}

func (hostMonController) StopSystem() error {
	return hostmon.Default().Stop()
}

func (hostMonController) SetSystemAutoStart(enabled bool) error {
	return hostmon.Default().SetAutoStart(enabled)
}

func formatPort(port int) string {
	if port <= 0 {
		return ""
	}
	return ":" + strconv.Itoa(port)
}
