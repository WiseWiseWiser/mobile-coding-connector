package portforward

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	connectorWatchdogTick    = 30 * time.Second
	connectorSlowThreshold   = 3 * time.Second
	connectorSlowStrikes     = 3
	connectorRecycleCooldown = 5 * time.Minute
	connectorProbeTimeout    = 5 * time.Second
	connectorStopWait        = 6 * time.Second
	// connectorBlackholeQueueBytes is the exact unacked send-queue size of a
	// blackholed edge connection: the kernel stops buffering at the default
	// 128KB send buffer when the peer never drains, and with no ACKs the
	// buffer never auto-tunes — the queue sits pinned at 131072 sweep after
	// sweep. Everything else is live traffic: streaming responses (the
	// live-activity feed) and client retry floods legitimately push queues
	// past the base cap (measured 130KB-1.6MB) while draining, so size-based
	// thresholds alone cannot tell an outage from a busy tunnel — only the
	// exact-cap pin can.
	connectorBlackholeQueueBytes = 131072
	// connectorBlackholeStrikes: consecutive exact-cap readings required
	// before recycling (three sweeps = 90s).
	connectorBlackholeStrikes = 3
)

// watchdogTunnelState is the per-tunnel bookkeeping (test-visible via reset).
type watchdogTunnelState struct {
	strikes          int
	blackholeStrikes int
	lastRecycle      time.Time
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
	// watchdogEdgeStallBytes reports the largest unacked send queue (bytes)
	// across the connector's edge (:7844) sockets, 0 when none are stalled.
	watchdogEdgeStallBytes = connectorEdgeStallBytes
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

// connectorEdgeStallBytes returns the largest unacked send queue across the
// given pid's established edge sockets, or 0. macOS-only tooling (lsof +
// netstat), matching the product's darwin target; errors read as "no signal"
// so a missing tool cannot cause recycles. Note lsof's -a: -p and -i are OR'd
// without it, which would attribute every TCP socket's queue to this pid.
func connectorEdgeStallBytes(pid int) int {
	if pid <= 0 {
		return 0
	}
	out, err := exec.Command("lsof", "-nP", "-a", "-p", strconv.Itoa(pid), "-i", "TCP").Output()
	if err != nil {
		return 0
	}
	localPorts := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		for _, f := range fields {
			if !strings.Contains(f, "->") {
				continue
			}
			parts := strings.SplitN(f, "->", 2)
			if !strings.HasSuffix(parts[1], ":7844") {
				continue
			}
			if _, port, err := net.SplitHostPort(parts[0]); err == nil {
				localPorts[port] = true
			}
		}
	}
	if len(localPorts) == 0 {
		return 0
	}
	nsOut, err := exec.Command("netstat", "-p", "tcp", "-n").Output()
	if err != nil {
		return 0
	}
	maxSendQ := 0
	for _, line := range strings.Split(string(nsOut), "\n") {
		fields := strings.Fields(line)
		// tcp4 <recv-q> <send-q> <local> <remote> <state>
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "tcp") {
			continue
		}
		local := fields[3]
		dot := strings.LastIndex(local, ".")
		if dot < 0 || !localPorts[local[dot+1:]] {
			continue
		}
		sendQ, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		if sendQ > maxSendQ {
			maxSendQ = sendQ
		}
	}
	return maxSendQ
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
		// Signal 1 (deterministic): a sustained unacked send queue on the
		// connector's edge sockets — the zombie signature. Catches partial
		// stalls that a sparse TTFB probe usually misses (the edge keeps most
		// requests on the healthy connections).
		stallBytes := 0
		if st.ConnectorPID > 0 && watchdogAlivePID(st.ConnectorPID) {
			stallBytes = watchdogEdgeStallBytes(st.ConnectorPID)
		}
		// Signal 2 (end-to-end): probe TTFB through the tunnel.
		ttfb, probeErr := watchdogProbeHost(watchdog.probeCli, host)
		watchdog.mu.Lock()
		book := watchdog.tunnels[key]
		if book == nil {
			book = &watchdogTunnelState{}
			watchdog.tunnels[key] = book
		}
		if probeErr != nil || ttfb <= connectorSlowThreshold {
			book.strikes = 0
		} else {
			book.strikes++
		}
		// A queue pinned at EXACTLY the base send-buffer cap across consecutive
		// sweeps is an active blackhole: the kernel stops at 131072 for a peer
		// that never drains and auto-tuning never kicks in without ACKs. Any
		// other large queue (686KB, 1.1MB, 1.6MB observed) is live streaming
		// traffic with a partially draining buffer — recycling it only killed
		// healthy connectors and looped the edge into 1033s. The exact-cap
		// signature needs three consecutive sweeps (90s), then bypasses the
		// cooldown — a confirmed outage must not wait out a cooldown.
		blackhole := stallBytes == connectorBlackholeQueueBytes
		if blackhole {
			book.blackholeStrikes++
		} else {
			book.blackholeStrikes = 0
		}
		slowed := book.strikes >= connectorSlowStrikes
		confirmedBlackhole := book.blackholeStrikes >= connectorBlackholeStrikes
		strikes := book.strikes
		inCooldown := !book.lastRecycle.IsZero() && now.Sub(book.lastRecycle) < connectorRecycleCooldown
		watchdog.mu.Unlock()

		if !slowed && !confirmedBlackhole {
			if strikes > 0 {
				fmt.Printf("[port mapping] connector slow: %s ttfb=%s strike %d/%d\n", host, ttfb.Round(time.Millisecond), strikes, connectorSlowStrikes)
			}
			if stallBytes > 0 {
				fmt.Printf("[port mapping] connector edge queue: pid=%d sendq=%d\n", st.ConnectorPID, stallBytes)
			}
			continue
		}
		if inCooldown && !confirmedBlackhole {
			continue
		}
		reason := fmt.Sprintf("ttfb=%s strikes=%d", ttfb.Round(time.Millisecond), strikes)
		if confirmedBlackhole {
			reason = fmt.Sprintf("edge sendq=%d blackhole %d/%d", stallBytes, book.blackholeStrikes, connectorBlackholeStrikes)
		}
		fmt.Printf("[port mapping] recycling stalled connector: tunnel=%s host=%s pid=%d (%s)\n", st.TunnelName, host, st.ConnectorPID, reason)
		if err := recycleConnector(st, cfgDir); err != nil {
			fmt.Printf("[port mapping] connector recycle failed: %v\n", err)
			continue
		}
		watchdog.mu.Lock()
		book.lastRecycle = now
		book.strikes = 0
		book.blackholeStrikes = 0
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
