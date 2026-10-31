package portforward

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	pkgcf "github.com/xhd2015/dot-pkgs/go-pkgs/cloudflare"
)

// The connector watchdog probes each managed host tunnel's public URL and
// recycles the cloudflared connector when the edge path stalls. Long-lived
// connector connections can decay into crawling TCP sessions that stay
// registered at the edge: neither cloudflared nor Cloudflare tears them down,
// so new requests keep landing on multi-second stalls while fresh connectors
// measure 100-300ms. Probing measures transport TTFB only; a 4xx body is a
// healthy probe. Probe errors (network down) reset the strike counter —
// recycling cannot fix an offline machine.

const (
	connectorWatchdogTick    = time.Minute
	connectorSlowThreshold   = 3 * time.Second
	connectorSlowStrikes     = 3
	connectorRecycleCooldown = 5 * time.Minute
	connectorProbeTimeout    = 5 * time.Second
	connectorStopWait        = 6 * time.Second
)

// watchdogTunnelState is the per-tunnel bookkeeping (test-visible via reset).
type watchdogTunnelState struct {
	strikes     int
	lastRecycle time.Time
}

type watchdogBook struct {
	mu       sync.Mutex
	tunnels  map[string]*watchdogTunnelState
	probeCli *http.Client
}

var watchdog = &watchdogBook{
	tunnels:  map[string]*watchdogTunnelState{},
	probeCli: &http.Client{Timeout: connectorProbeTimeout},
}

// Injectable seams (mirrors reconcileOwnedConnectors in restore_connectors.go).
var (
	watchdogConfigDir = func() (string, error) { return pkgcf.DefaultConfigDir() }
	watchdogProbeHost = func(cli *http.Client, host string) (time.Duration, error) {
		start := time.Now()
		resp, err := cli.Get(fmt.Sprintf("https://%s/", host))
		if err != nil {
			return 0, err
		}
		resp.Body.Close()
		return time.Since(start), nil
	}
	watchdogStopPID  = stopConnectorPID
	watchdogAttach   = attachManagedTunnel
	watchdogNow      = time.Now
	watchdogAlivePID = pidAlive
)

// attachManagedTunnel re-runs the host attach flow for one host. Attach loads
// the full tunnel state, rewrites config.yml from every host, and starts a
// fresh connector when the recorded PID is dead — the same path
// backend/host.go uses, so the new connector carries the pinned protocol.
func attachManagedTunnel(st *pkgcf.TunnelState, configDir, host, service string) error {
	_, err := pkgcf.Attach(pkgcf.AttachOptions{
		Domain:     host,
		LocalURL:   service,
		TunnelName: st.TunnelName,
		ConfigDir:  configDir,
		OwnerPID:   os.Getpid(),
	})
	return err
}

// stopConnectorPID SIGTERMs the connector process group, waits, then SIGKILLs.
// Connectors are spawned with Setpgid (pgid = pid), matching dot-pkgs Stop.
func stopConnectorPID(pid int) {
	if pid <= 0 {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	_ = syscall.Kill(pid, syscall.SIGTERM)
	deadline := time.Now().Add(connectorStopWait)
	for pidAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if !pidAlive(pid) {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

// recycleConnector stops the stalled connector and re-attaches. If a live
// supervisor respawns first, Attach adopts it; otherwise Attach starts a fresh
// connector via startRegisteredConnector (which also stops any supervisor).
func recycleConnector(st *pkgcf.TunnelState, configDir string) error {
	watchdogStopPID(st.ConnectorPID)
	host, service, ok := preferredWatchdogHost(st)
	if !ok {
		return errors.New("no host to re-attach")
	}
	if err := watchdogAttach(st, configDir, host, service); err != nil {
		return fmt.Errorf("re-attach %s: %w", host, err)
	}
	return nil
}

// preferredWatchdogHost picks a deterministic host: sorted-first hostname.
func preferredWatchdogHost(st *pkgcf.TunnelState) (string, string, bool) {
	hosts := make([]string, 0, len(st.Hosts))
	for host := range st.Hosts {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	for _, host := range hosts {
		entry := st.Hosts[host]
		if entry == nil || strings.TrimSpace(entry.Service) == "" {
			continue
		}
		return host, entry.Service, true
	}
	return "", "", false
}

// connectorWatchdogSweep runs one probe+maybe-recycle pass. Exposed for tests.
func connectorWatchdogSweep() {
	cfgDir, err := watchdogConfigDir()
	if err != nil {
		return
	}
	root := filepath.Join(cfgDir, "managed-tunnels")
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	now := watchdogNow()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		st, err := pkgcf.LoadTunnelState(dir)
		if err != nil || st == nil || len(st.Hosts) == 0 {
			continue
		}
		host, _, ok := preferredWatchdogHost(st)
		if !ok {
			continue
		}
		key := dir
		ttfb, probeErr := watchdogProbeHost(watchdog.probeCli, host)
		watchdog.mu.Lock()
		book := watchdog.tunnels[key]
		if book == nil {
			book = &watchdogTunnelState{}
			watchdog.tunnels[key] = book
		}
		if probeErr != nil || ttfb <= connectorSlowThreshold {
			book.strikes = 0
			watchdog.mu.Unlock()
			continue
		}
		book.strikes++
		strikes := book.strikes
		inCooldown := !book.lastRecycle.IsZero() && now.Sub(book.lastRecycle) < connectorRecycleCooldown
		watchdog.mu.Unlock()

		if strikes < connectorSlowStrikes {
			fmt.Printf("[port mapping] connector slow: %s ttfb=%s strike %d/%d\n", host, ttfb.Round(time.Millisecond), strikes, connectorSlowStrikes)
			continue
		}
		if inCooldown {
			continue
		}
		fmt.Printf("[port mapping] recycling stalled connector: tunnel=%s host=%s pid=%d (ttfb=%s, %d strikes)\n", st.TunnelName, host, st.ConnectorPID, ttfb.Round(time.Millisecond), strikes)
		if err := recycleConnector(st, cfgDir); err != nil {
			fmt.Printf("[port mapping] connector recycle failed: %v\n", err)
			continue
		}
		watchdog.mu.Lock()
		book.lastRecycle = now
		book.strikes = 0
		watchdog.mu.Unlock()
		fmt.Printf("[port mapping] connector recycled: tunnel=%s\n", st.TunnelName)
	}
}

// StartConnectorWatchdog probes managed tunnels once per tick and recycles
// connectors that stay slow. Safe to call multiple times: it starts at most
// one goroutine per process.
func StartConnectorWatchdog() {
	watchdogStartOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(connectorWatchdogTick)
			defer ticker.Stop()
			for range ticker.C {
				connectorWatchdogSweep()
			}
		}()
	})
}

var watchdogStartOnce sync.Once

func resetWatchdogForTest() {
	watchdog.mu.Lock()
	defer watchdog.mu.Unlock()
	watchdog.tunnels = map[string]*watchdogTunnelState{}
}
