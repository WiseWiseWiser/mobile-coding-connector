package domains

import (
	"context"
	"fmt"
	"sync"
	"time"

	cloudflareSettings "github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
)

// domainReconciler owns configured bootstrap hostnames. The selected backend is
// the only publisher: a second core-group mapping must not overwrite its DNS.
// Retry state is process-local; server-domains.json remains the desired config.
type domainReconciler struct {
	mu                                         sync.Mutex
	slots                                      map[string]*domainSlot
	closed                                     bool
	configCancel                               context.CancelFunc
	loadConfig                                 func() (*DomainsConfig, error)
	start                                      func(string, int, cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error)
	retry                                      func(string, int, cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error)
	stop                                       func(string, cloudflareSettings.LogFunc) error
	probe                                      func(string) (int, error)
	configured                                 func(string) (bool, error)
	retryMin, retryMax, probeInterval          time.Duration
	failuresBeforeRetry, connectingBeforeRetry int
}

type domainSlot struct {
	mu              sync.Mutex
	opMu            sync.Mutex // serializes start and stop, including in-flight starts
	generation      uint64
	cancel          context.CancelFunc
	desired         bool
	manuallyStopped bool
	status          cloudflareSettings.DomainTunnelStatus
}

func newDomainReconciler() *domainReconciler {
	return &domainReconciler{
		slots:      make(map[string]*domainSlot),
		loadConfig: LoadDomains,
		start:      startBootstrapDomainTunnel,
		retry:      retryBootstrapDomainTunnel,
		stop:       StopHostDomainTunnel,
		probe:      unified_tunnel.ProbeHostname,
		configured: configuredBootstrapDomain,
		retryMin:   2 * time.Second, retryMax: 30 * time.Second, probeInterval: 10 * time.Second,
		failuresBeforeRetry: 3, connectingBeforeRetry: 6,
	}
}

var bootstrapDomains = newDomainReconciler()

func getBootstrapDomainStatus(domain string) cloudflareSettings.DomainTunnelStatus {
	if status, ok := bootstrapDomains.status(domain); ok {
		return status
	}
	return cloudflareSettings.GetDomainTunnelStatus(domain)
}

func configuredBootstrapDomain(domain string) (bool, error) {
	cfg, err := LoadDomains()
	if err != nil {
		return false, err
	}
	for _, d := range cfg.Domains {
		if d.Domain == domain && d.Provider == ProviderCloudflare {
			return true, nil
		}
	}
	return false, nil
}

func (r *domainReconciler) slot(domain string) *domainSlot {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.slots == nil {
		r.slots = make(map[string]*domainSlot)
	}
	s := r.slots[domain]
	if s == nil {
		s = &domainSlot{}
		r.slots[domain] = s
	}
	return s
}

func domainLogBuffer(domain string) *healthCheckLogBuffer {
	healthCheckLogsMu.Lock()
	defer healthCheckLogsMu.Unlock()
	b := healthCheckLogs[domain]
	if b == nil {
		b = newHealthCheckLogBuffer()
		healthCheckLogs[domain] = b
	}
	return b
}

func (r *domainReconciler) log(domain, text string) {
	domainLogBuffer(domain).addLog(text)
	fmt.Printf("[domains] %s: %s\n", domain, text)
}

func (s *domainSlot) replace(domain string, onlyIfAbsent bool) (context.Context, uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if onlyIfAbsent && (s.manuallyStopped || (s.desired && s.cancel != nil)) {
		return nil, 0, false
	}
	if !onlyIfAbsent {
		s.manuallyStopped = false
	}
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.generation++
	s.cancel = cancel
	s.desired = true
	s.status = cloudflareSettings.DomainTunnelStatus{Status: "connecting", TunnelURL: "https://" + domain}
	return ctx, s.generation, true
}

func (s *domainSlot) current(ctx context.Context, generation uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ctx.Err() == nil && s.desired && s.generation == generation
}

func (s *domainSlot) update(ctx context.Context, generation uint64, status cloudflareSettings.DomainTunnelStatus) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil || !s.desired || s.generation != generation {
		return false
	}
	s.status = status
	return true
}

func (r *domainReconciler) status(domain string) (cloudflareSettings.DomainTunnelStatus, bool) {
	r.mu.Lock()
	s := r.slots[domain]
	r.mu.Unlock()
	if s == nil {
		return cloudflareSettings.DomainTunnelStatus{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status, true
}

func (r *domainReconciler) startAutomatic(domain string, port int) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	s := r.slots[domain]
	if s == nil {
		s = &domainSlot{}
		r.slots[domain] = s
	}
	ctx, gen, ok := s.replace(domain, true)
	r.mu.Unlock()
	if ok {
		go r.run(ctx, s, gen, domain, port, false, false)
	}
}

// A transient failure reading the desired config must not disable bootstrap.
func (r *domainReconciler) startConfigured(port int) {
	r.mu.Lock()
	if r.closed || r.configCancel != nil {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.configCancel = cancel
	r.mu.Unlock()
	go func() {
		var backoff time.Duration
		for ctx.Err() == nil {
			cfg, err := r.loadConfig()
			if err == nil {
				for _, d := range cfg.Domains {
					if ctx.Err() != nil {
						return
					}
					if d.Provider == ProviderCloudflare {
						r.startAutomatic(d.Domain, port)
					}
				}
				return
			}
			backoff = r.nextBackoff(backoff)
			fmt.Printf("[domains] auto-start: load config: %v; retrying in %s\n", err, backoff)
			if !waitDomain(ctx, backoff) {
				return
			}
		}
	}()
}

func (r *domainReconciler) attempt(ctx context.Context, s *domainSlot, gen uint64, domain string, port int, logFn cloudflareSettings.LogFunc, retry bool) (*cloudflareSettings.DomainTunnelStatus, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if !s.current(ctx, gen) {
		return nil, context.Canceled
	}
	if retry && r.retry != nil {
		return r.retry(domain, port, logFn)
	}
	return r.start(domain, port, logFn)
}

// startManual cancels the prior worker, serializes with any in-flight attempt,
// and leaves a retrying worker behind even if this explicit first attempt fails.
func (r *domainReconciler) startManual(domain string, port int, logFn cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, context.Canceled
	}
	s := r.slots[domain]
	if s == nil {
		s = &domainSlot{}
		r.slots[domain] = s
	}
	ctx, gen, _ := s.replace(domain, false)
	r.mu.Unlock()
	// Proxy publishers may continue logging after the HTTP handler returns.
	// Never write to its SSE response after the initial attempt completes.
	var streamMu sync.Mutex
	streaming := true
	forwardLog := func(message string) {
		r.log(domain, message)
		streamMu.Lock()
		defer streamMu.Unlock()
		if streaming && logFn != nil {
			logFn(message)
		}
	}
	status, err := r.attempt(ctx, s, gen, domain, port, forwardLog, false)
	streamMu.Lock()
	streaming = false
	streamMu.Unlock()
	if !s.current(ctx, gen) {
		return nil, context.Canceled
	}
	if err != nil {
		r.setError(ctx, s, gen, domain, err)
		go r.run(ctx, s, gen, domain, port, false, true)
		return nil, err
	}
	r.log(domain, "tunnel started; checking public reachability")
	go r.run(ctx, s, gen, domain, port, true, false)
	return status, nil
}

func (r *domainReconciler) stopDomain(domain string, logFn cloudflareSettings.LogFunc) error {
	s := r.slot(domain)
	s.mu.Lock()
	s.desired = false
	s.manuallyStopped = true
	s.generation++
	generation := s.generation
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.status = cloudflareSettings.DomainTunnelStatus{Status: "stopped"}
	s.mu.Unlock()
	// A stop waits for any in-flight start before detaching. A canceled worker
	// cannot start again after this returns, including during a network outage.
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	superseded := s.generation != generation || s.desired
	s.mu.Unlock()
	if superseded {
		return context.Canceled
	}
	err := r.stop(domain, logFn)
	if err != nil {
		s.mu.Lock()
		if s.generation == generation && !s.desired {
			s.status = cloudflareSettings.DomainTunnelStatus{Status: "error", TunnelURL: "https://" + domain, Error: "stop failed: " + err.Error()}
		}
		s.mu.Unlock()
		r.log(domain, "stop failed: "+err.Error())
	}
	return err
}

func (r *domainReconciler) shutdown() {
	r.mu.Lock()
	r.closed = true
	if r.configCancel != nil {
		r.configCancel()
		r.configCancel = nil
	}
	slots := make([]*domainSlot, 0, len(r.slots))
	for _, s := range r.slots {
		slots = append(slots, s)
	}
	r.mu.Unlock()
	for _, s := range slots {
		s.mu.Lock()
		s.desired = false
		s.generation++
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
		s.mu.Unlock()
	}
}

func (r *domainReconciler) setError(ctx context.Context, s *domainSlot, gen uint64, domain string, err error) {
	if s.update(ctx, gen, cloudflareSettings.DomainTunnelStatus{Status: "error", TunnelURL: "https://" + domain, Error: err.Error()}) {
		r.log(domain, err.Error())
	}
}

func waitDomain(ctx context.Context, delay time.Duration) bool {
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func (r *domainReconciler) nextBackoff(delay time.Duration) time.Duration {
	if delay <= 0 {
		return r.retryMin
	}
	if delay >= r.retryMax/2 {
		return r.retryMax
	}
	return delay * 2
}

func (r *domainReconciler) run(ctx context.Context, s *domainSlot, gen uint64, domain string, port int, ready, delayFirst bool) {
	var backoff time.Duration
	forceRetry := false
	failures, connecting := 0, 0
	if delayFirst {
		backoff = r.nextBackoff(backoff)
		if !waitDomain(ctx, backoff) {
			return
		}
	}
	for s.current(ctx, gen) {
		configured, err := r.configured(domain)
		if err == nil && !configured {
			s.mu.Lock()
			if s.generation == gen {
				s.desired = false
				s.cancel = nil
				s.status = cloudflareSettings.DomainTunnelStatus{Status: "stopped"}
			}
			s.mu.Unlock()
			r.log(domain, "domain no longer configured; stopped retrying")
			return
		}
		if err != nil {
			r.setError(ctx, s, gen, domain, fmt.Errorf("load desired domain: %w", err))
			backoff = r.nextBackoff(backoff)
			if !waitDomain(ctx, backoff) {
				return
			}
			continue
		}
		if !ready {
			s.update(ctx, gen, cloudflareSettings.DomainTunnelStatus{Status: "connecting", TunnelURL: "https://" + domain})
			_, err := r.attempt(ctx, s, gen, domain, port, func(msg string) { r.log(domain, msg) }, forceRetry)
			if !s.current(ctx, gen) {
				return
			}
			if err != nil {
				r.setError(ctx, s, gen, domain, fmt.Errorf("auto-start failed: %w", err))
				backoff = r.nextBackoff(backoff)
				r.log(domain, fmt.Sprintf("retrying in %s", backoff))
				if !waitDomain(ctx, backoff) {
					return
				}
				continue
			}
			r.log(domain, "tunnel started; checking public reachability")
			ready = true
			forceRetry = false
			backoff = 0
			failures, connecting = 0, 0
		}
		code, err := r.probe(domain)
		if !s.current(ctx, gen) {
			return
		}
		switch {
		case err == nil && unified_tunnel.IsEdgeRegistered(code):
			if s.update(ctx, gen, cloudflareSettings.DomainTunnelStatus{Status: "active", TunnelURL: "https://" + domain}) && (failures > 0 || connecting > 0) {
				r.log(domain, "public domain recovered")
			}
			failures, connecting = 0, 0
		case err == nil && code == unified_tunnel.CloudflareErrorOriginDown:
			s.update(ctx, gen, cloudflareSettings.DomainTunnelStatus{Status: "connecting", TunnelURL: "https://" + domain})
			connecting++
			failures = 0
			// Do not kill a connecting connector. A later idempotent reattach will
			// also restart a dead managed connector without touching siblings.
			if connecting >= r.connectingBeforeRetry {
				ready = false
				connecting = 0
				r.log(domain, "still connecting; rechecking desired route")
			}
		default:
			if err == nil {
				err = fmt.Errorf("public health check returned HTTP %d", code)
			}
			failures++
			r.setError(ctx, s, gen, domain, fmt.Errorf("public health check failed: %w", err))
			if failures >= r.failuresBeforeRetry {
				ready = false
				forceRetry = true
				failures = 0
				r.log(domain, "rechecking desired route after repeated failures")
			}
		}
		if !waitDomain(ctx, r.probeInterval) {
			return
		}
	}
}
