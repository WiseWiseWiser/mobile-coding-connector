package server

import (
	"fmt"
	"time"

	opencode_exposed "github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode/exposed_opencode"
	"github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare/unified_tunnel"
	"github.com/xhd2015/lifelog-private/ai-critic/server/domains"
	"github.com/xhd2015/lifelog-private/ai-critic/server/exposedurls"
	"github.com/xhd2015/lifelog-private/ai-critic/server/services"
)

func RunBackgroundTasks() {
	fmt.Printf("[auto-task] Running background tasks\n")
	opencode_exposed.StartHealthCheck()
	unified_tunnel.StartGlobalHealthChecks()
	services.StartHealthCheck()
}

func RunStartupTasks() {
	fmt.Printf("[auto-task] Running startup tasks\n")
	domains.AutoStartTunnels()
	opencode_exposed.AutoStartWebServer()
	services.AutoStartConfiguredServices()

	go func() {
		time.Sleep(2 * time.Second)
		domains.InitDomainTunnels()
		exposedurls.InitExposedURLTunnels()
	}()
}

func RunSideEffectTasks() {
	fmt.Printf("[auto-task] Running side effect tasks\n")
	RunBackgroundTasks()
	RunStartupTasks()
}
