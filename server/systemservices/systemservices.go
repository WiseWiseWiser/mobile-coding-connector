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
	"strconv"
	"strings"

	"github.com/xhd2015/ai-critic/server/gomod"
	"github.com/xhd2015/ai-critic/server/openclaw"
	"github.com/xhd2015/ai-critic/server/proxy/wsproxy"
	"github.com/xhd2015/ai-critic/server/services"
)

// Stable ids: the CLI and the GUI address system services by them. The sys-
// prefix keeps them clear of the generated svc- ids.
const (
	WSProxyID  = "sys-ws-proxy"
	CFProxyID  = "sys-cloudflare-proxy"
	GoModID    = "sys-go-mod-proxy"
	OpenClawID = "sys-openclaw"
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
	_ services.SystemController   = wsProxyController{}
	_ services.SystemLifecycle    = wsProxyController{}
	_ services.SystemAutoStarter  = wsProxyController{}
	_ services.SystemController   = goModController{}
	_ services.SystemLifecycle    = goModController{}
	_ services.SystemAutoStarter  = goModController{}
	_ services.SystemController   = openClawController{}
	_ services.SystemLifecycle    = openClawController{}
	_ services.SystemAutoStarter  = openClawController{}
	_ services.SystemController   = &cloudFlareProxyController{}
	_ services.SystemRestarter    = &cloudFlareProxyController{}
	_ services.SystemHostProvider = &cloudFlareProxyController{}
)

func boolPtr(v bool) *bool { return &v }

// wsProxyController wraps the in-process Xray/tunnel proxy.
type wsProxyController struct{}

func (wsProxyController) SystemStatus() services.SystemRuntime {
	runtime := services.SystemRuntime{}

	// LoadConfig falls back to defaults when nothing was saved, and the derived
	// public URL then embeds a freshly generated random instance id. Acting on
	// that would persist junk (and change the published hostname), so an
	// unconfigured proxy exposes no auto-start switch and no public URL.
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
	return wsproxy.GetManager().Start(false)
}

func (wsProxyController) StopSystem() error {
	return wsproxy.GetManager().Stop()
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

func formatPort(port int) string {
	if port <= 0 {
		return ""
	}
	return ":" + strconv.Itoa(port)
}
