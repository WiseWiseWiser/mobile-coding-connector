package server

import (
	"fmt"
	"time"

	opencode_exposed "github.com/xhd2015/ai-critic/server/agents/opencode/exposed_opencode"
	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
	"github.com/xhd2015/ai-critic/server/domains"
	"github.com/xhd2015/ai-critic/server/exposedurls"
	"github.com/xhd2015/ai-critic/server/proxy/wsproxy"
	"github.com/xhd2015/ai-critic/server/services"
	"github.com/xhd2015/ai-critic/server/startup"
)

func RunBackgroundTasks() {
	fmt.Printf("[auto-task] Running background tasks\n")
	opencode_exposed.StartHealthCheck()
	unified_tunnel.StartGlobalHealthChecks()
	services.StartHealthCheck()
}

func runExtensionWork() {
	domains.AutoStartTunnels()
	opencode_exposed.AutoStartWebServer()
	services.AutoStartConfiguredServices()
	wsproxy.AutoStart()

	go func() {
		time.Sleep(2 * time.Second)
		domains.InitDomainTunnels()
		exposedurls.InitExposedURLTunnels()
	}()
}

// RunCoreStartup runs synchronous, minimal startup (background health checks).
func RunCoreStartup() {
	RunBackgroundTasks()
}

// RunExtensionStartup runs I/O-heavy extension work; safe to call in a goroutine.
func RunExtensionStartup() {
	if startup.SkipExtensionStartup() {
		logBootstrapPhase("extension_done", 0, "err=skipped")
		return
	}
	if delay := startup.ExtensionStartupDelay(); delay > 0 {
		time.Sleep(delay)
	}
	logBootstrapPhase("extension_start", 0, "")
	fmt.Printf("[auto-task] Running extension\n")
	runExtensionWork()
	logBootstrapPhase("extension_done", 0, "err=nil")
}

func RunStartupTasks() {
	fmt.Printf("[auto-task] Running startup tasks\n")
	runExtensionWork()
}

func RunSideEffectTasks() {
	fmt.Printf("[auto-task] Running side effect tasks\n")
	RunCoreStartup()
	RunExtensionStartup()
}