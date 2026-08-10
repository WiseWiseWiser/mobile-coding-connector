package eventbus

import (
	"fmt"
	"net/http"
	"sync"
)

// Process-level wiring for the main ai-critic server. Injectable for L2 tests
// via package constructors; these helpers only configure the running binary.

var (
	lifecycleMu     sync.Mutex
	defaultHub      *Hub
	publishCfg      PublishConfig
	publishSrv      *PublishServer
	cfgInitialized  bool
)

// SetPublishConfig stores the resolved flag config used at Serve startup.
// Call before StartFromConfig / RegisterDefault.
func SetPublishConfig(cfg PublishConfig) {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	publishCfg = cfg
	cfgInitialized = true
}

// GetPublishConfig returns the configured publish settings (defaults if unset).
func GetPublishConfig() PublishConfig {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	if !cfgInitialized {
		return ResolvePublishConfig(0, "", false)
	}
	return publishCfg
}

// DefaultHub returns (and lazily creates) the process-wide hub used by WS + publish.
func DefaultHub() *Hub {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	if defaultHub == nil {
		defaultHub = NewHub(200)
	}
	return defaultHub
}

// RegisterDefault registers the WS subscribe path on the main mux using DefaultHub.
func RegisterDefault(mux *http.ServeMux) {
	RegisterSubscribeWS(mux, DefaultHub())
}

// StartFromConfig starts the loopback publish server when not disabled.
// Hard-fails if the configured port is already in use.
// Safe to call once per process; subsequent calls no-op if already started.
func StartFromConfig() error {
	lifecycleMu.Lock()
	cfg := publishCfg
	if !cfgInitialized {
		cfg = ResolvePublishConfig(0, "", false)
		publishCfg = cfg
		cfgInitialized = true
	}
	if cfg.Disabled {
		lifecycleMu.Unlock()
		return nil
	}
	if publishSrv != nil {
		lifecycleMu.Unlock()
		return nil
	}
	hub := defaultHub
	if hub == nil {
		hub = NewHub(200)
		defaultHub = hub
	}
	lifecycleMu.Unlock()

	srv, err := StartPublishServer(cfg.ListenAddr(), hub, PublishServerOpts{Token: cfg.Token})
	if err != nil {
		return fmt.Errorf("event-bus publish server on %s: %w", cfg.ListenAddr(), err)
	}

	lifecycleMu.Lock()
	publishSrv = srv
	lifecycleMu.Unlock()
	return nil
}

// StopPublishServer closes the process-level publish listener if running.
func StopPublishServer() {
	lifecycleMu.Lock()
	srv := publishSrv
	publishSrv = nil
	lifecycleMu.Unlock()
	if srv != nil {
		_ = srv.Close()
	}
}
