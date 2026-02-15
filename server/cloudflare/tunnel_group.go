package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

type TunnelGroup struct {
	mu        sync.RWMutex
	name      string
	tunnelMgr *UnifiedTunnelManager

	paused                 bool
	healthCheckPausedUntil map[string]time.Time

	healthCtx    context.Context
	healthCancel context.CancelFunc

	onHealthChange MappingHealthCallback
}

func NewTunnelGroup(name string, tunnelMgr *UnifiedTunnelManager) *TunnelGroup {
	return &TunnelGroup{
		name:                   name,
		tunnelMgr:              tunnelMgr,
		healthCheckPausedUntil: make(map[string]time.Time),
	}
}

func (tg *TunnelGroup) Name() string {
	return tg.name
}

func (tg *TunnelGroup) AddMapping(mapping *IngressMapping) error {
	fmt.Printf("[tunnel-group:%s] AddMapping: id=%s hostname=%s service=%s\n", tg.name, mapping.ID, mapping.Hostname, mapping.Service)
	return tg.tunnelMgr.AddMapping(mapping)
}

func (tg *TunnelGroup) RemoveMapping(id string) error {
	fmt.Printf("[tunnel-group:%s] RemoveMapping: id=%s\n", tg.name, id)
	return tg.tunnelMgr.RemoveMapping(id)
}

func (tg *TunnelGroup) ListMappings() []*IngressMapping {
	return tg.tunnelMgr.ListMappings()
}

func (tg *TunnelGroup) GetMapping(mappingID string) (*IngressMapping, bool) {
	return tg.tunnelMgr.GetMapping(mappingID)
}

func (tg *TunnelGroup) IsRunning() bool {
	return tg.tunnelMgr.IsRunning()
}

func (tg *TunnelGroup) GetStatus() map[string]interface{} {
	return tg.tunnelMgr.GetTunnelStatus()
}

func (tg *TunnelGroup) PauseHealthCheck(mappingID string, duration time.Duration) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	pauseUntil := time.Now().Add(duration)
	tg.healthCheckPausedUntil[mappingID] = pauseUntil
	fmt.Printf("[tunnel-group:%s] PauseHealthCheck: paused health checks for mapping %s until %v\n",
		tg.name, mappingID, pauseUntil.Format("2006-01-02T15:04:05"))
}

func (tg *TunnelGroup) IsHealthCheckPaused(mappingID string) bool {
	tg.mu.RLock()
	defer tg.mu.RUnlock()

	if tg.paused {
		return true
	}

	pauseUntil, exists := tg.healthCheckPausedUntil[mappingID]
	if !exists {
		return false
	}

	return time.Now().Before(pauseUntil)
}

func (tg *TunnelGroup) RestartMapping(mappingID string) error {
	fmt.Printf("[tunnel-group:%s] RestartMapping: triggering restart for mappingID=%s\n", tg.name, mappingID)

	tg.mu.Lock()
	_, exists := tg.tunnelMgr.mappings[mappingID]
	if !exists {
		tg.mu.Unlock()
		return fmt.Errorf("mapping %s not found", mappingID)
	}
	tg.mu.Unlock()

	err := tg.tunnelMgr.RestartMapping(mappingID)

	if err == nil {
		tg.PauseHealthCheck(mappingID, 1*time.Minute)
	}

	return err
}

func (tg *TunnelGroup) StartHealthChecks(callback MappingHealthCallback) {
	tg.onHealthChange = callback
	tg.healthCtx, tg.healthCancel = context.WithCancel(context.Background())

	go func() {
		type healthState struct {
			consecutiveFailures int
			lastHealthy         bool
		}

		states := make(map[string]*healthState)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		select {
		case <-time.After(5 * time.Second):
		case <-tg.healthCtx.Done():
			return
		}

		for {
			select {
			case <-tg.healthCtx.Done():
				return
			case <-ticker.C:
				tg.mu.RLock()
				paused := tg.paused
				mappings := make([]*IngressMapping, 0, len(tg.tunnelMgr.mappings))
				for _, m := range tg.tunnelMgr.mappings {
					mappings = append(mappings, m)
				}
				tg.mu.RUnlock()

				if paused {
					fmt.Printf("[tunnel-group:%s] StartHealthChecks: health checks paused, skipping\n", tg.name)
					continue
				}

				fmt.Printf("[tunnel-group:%s] StartHealthChecks: checking %d mappings\n", tg.name, len(mappings))
				for _, m := range mappings {
					tg.mu.RLock()
					pauseUntil, isPaused := tg.healthCheckPausedUntil[m.ID]
					tg.mu.RUnlock()

					now := time.Now()
					if isPaused && now.Before(pauseUntil) {
						fmt.Printf("[tunnel-group:%s] StartHealthChecks: skipping paused mapping id=%s hostname=%s (paused until %v)\n",
							tg.name, m.ID, m.Hostname, pauseUntil.Format("2006-01-02T15:04:05"))
						continue
					}

					if isPaused && now.After(pauseUntil) {
						tg.mu.Lock()
						delete(tg.healthCheckPausedUntil, m.ID)
						tg.mu.Unlock()

						state, exists := states[m.ID]
						if exists {
							state.consecutiveFailures = 0
						}
					}

					fmt.Printf("[tunnel-group:%s] StartHealthChecks: checking mapping id=%s hostname=%s\n", tg.name, m.ID, m.Hostname)
					healthy := tg.checkMappingHealth(m.Hostname)

					state, exists := states[m.ID]
					if !exists {
						state = &healthState{lastHealthy: true}
						states[m.ID] = state
					}

					if healthy {
						if !state.lastHealthy {
							state.consecutiveFailures = 0
							state.lastHealthy = true
							if callback != nil {
								callback(m.ID, m.Hostname, true, 0)
							}
						}
					} else {
						state.consecutiveFailures++
						state.lastHealthy = false
						if callback != nil {
							callback(m.ID, m.Hostname, false, state.consecutiveFailures)
						}
					}
				}
			}
		}
	}()

	fmt.Printf("[tunnel-group:%s] Health checks started\n", tg.name)
}

func (tg *TunnelGroup) StopHealthChecks() {
	if tg.healthCancel != nil {
		tg.healthCancel()
		tg.healthCancel = nil
		fmt.Printf("[tunnel-group:%s] Health checks stopped\n", tg.name)
	}
}

func (tg *TunnelGroup) checkMappingHealth(hostname string) bool {
	fmt.Printf("[tunnel-group:%s] checkMappingHealth: checking health for hostname=%s\n", tg.name, hostname)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	urls := []string{
		fmt.Sprintf("https://%s/", hostname),
		fmt.Sprintf("https://%s/ping", hostname),
	}

	for _, url := range urls {
		fmt.Printf("[tunnel-group:%s] checkMappingHealth: trying %s\n", tg.name, url)
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("[tunnel-group:%s] checkMappingHealth: %s failed: %v\n", tg.name, url, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			fmt.Printf("[tunnel-group:%s] checkMappingHealth: %s returned status %d, healthy=true\n", tg.name, url, resp.StatusCode)
			return true
		}
		fmt.Printf("[tunnel-group:%s] checkMappingHealth: %s returned status %d, unhealthy\n", tg.name, url, resp.StatusCode)
	}

	fmt.Printf("[tunnel-group:%s] checkMappingHealth: all URLs failed for %s, marking unhealthy\n", tg.name, hostname)
	return false
}

func (tg *TunnelGroup) SetConfig(cfg config.CloudflareTunnelConfig) {
	tg.tunnelMgr.SetConfig(cfg)
}

func (tg *TunnelGroup) GetConfig() *config.CloudflareTunnelConfig {
	return tg.tunnelMgr.GetConfig()
}

func (tg *TunnelGroup) ListAllMappings() []*IngressMapping {
	return tg.tunnelMgr.ListAllMappings()
}

func (tg *TunnelGroup) GetConfigPath() string {
	return tg.tunnelMgr.GetConfigPath()
}

func (tg *TunnelGroup) GetExtraMappingsPath() string {
	return tg.tunnelMgr.GetExtraMappingsPath()
}

func (tg *TunnelGroup) LoadExtraMappingsFile() (*ExtraMappingsConfig, error) {
	return tg.tunnelMgr.LoadExtraMappingsFile()
}

func (tg *TunnelGroup) SaveExtraMappingsFile(cfg *ExtraMappingsConfig) error {
	return tg.tunnelMgr.SaveExtraMappingsFile(cfg)
}
