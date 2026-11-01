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
	// connectorStallBytes is the unacked send-queue size at which an edge
	// connection counts as stalled. Control-plane traffic is tiny, so a
	// sustained queue means the TCP path is blackholing while the connection
	// stays registered and keeps receiving requests.
	connectorStallBytes = 32 * 1024
	// connectorStallStrikes: stalled sweeps required before recycling. One
	// sweep can catch a transient upload; a real stall persists for minutes.
	connectorStallStrikes = 2
	// connectorSevereStallBytes: a queue at-or-above the socket send-buffer
	// cap (observed 128KB = full blackhole). A SINGLE reading is not evidence:
	// a fresh connector hit by a client retry flood — or any streaming
	// response (the live-activity feed) on a cold connection — buffers past
	// the base cap and drains within a tick or two (measured 130KB-1.1MB
	// bursts). Only a blackhole stays at-or-above the cap sweep after sweep,
	// so severe needs three consecutive readings, but then bypasses the
	// cooldown (a cooldown must not extend a confirmed outage; overnight it
	// stalled recycles for minutes: strike 5 at the same full queue).
	connectorSevereStallBytes = 120 * 1024
	connectorSevereStrikes    = 3
)

// watchdogTunnelState is the per-tunnel bookkeeping (test-visible via reset).
type watchdogTunnelState struct {
	strikes       int
	stallStrikes  int
	severeStrikes int
	lastRecycle   time.Time
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
		if stallBytes > connectorStallBytes {
			book.stallStrikes++
		} else {
			book.stallStrikes = 0
		}
		// A queue pinned at the send-buffer cap across consecutive sweeps is an
		// active blackhole. A SINGLE reading is not: a fresh connector hit by a
		// client retry flood legitimately buffers past the base cap while its
		// slow-start window opens, and the queue drains within a tick (that
		// single-shot rule recycle-looped the connector every tick). Severe
		// therefore needs two consecutive pinned readings, but bypasses the
		// cooldown — the cooldown must not extend a confirmed outage.
		if stallBytes >= connectorSevereStallBytes {
			book.severeStrikes++
		} else {
			book.severeStrikes = 0
		}
		slowed := book.strikes >= connectorSlowStrikes
		stalled := book.stallStrikes >= connectorStallStrikes
		severe := book.severeStrikes >= connectorSevereStrikes
		strikes := book.strikes
		inCooldown := !book.lastRecycle.IsZero() && now.Sub(book.lastRecycle) < connectorRecycleCooldown
		watchdog.mu.Unlock()

		if !slowed && !stalled && !severe {
			if strikes > 0 {
				fmt.Printf("[port mapping] connector slow: %s ttfb=%s strike %d/%d\n", host, ttfb.Round(time.Millisecond), strikes, connectorSlowStrikes)
			}
			if stallBytes > 0 {
				fmt.Printf("[port mapping] connector edge queue: pid=%d sendq=%d strike %d/%d\n", st.ConnectorPID, stallBytes, book.stallStrikes, connectorStallStrikes)
			}
			continue
		}
		if inCooldown && !severe {
			continue
		}
		reason := fmt.Sprintf("ttfb=%s strikes=%d", ttfb.Round(time.Millisecond), strikes)
		if stalled {
			reason = fmt.Sprintf("edge sendq=%d stallStrikes=%d", stallBytes, book.stallStrikes)
		}
		if severe {
			reason = fmt.Sprintf("edge sendq=%d severe blackhole %d/%d", stallBytes, book.severeStrikes, connectorSevereStrikes)
		}
		fmt.Printf("[port mapping] recycling stalled connector: tunnel=%s host=%s pid=%d (%s)\n", st.TunnelName, host, st.ConnectorPID, reason)
		if err := recycleConnector(st, cfgDir); err != nil {
			fmt.Printf("[port mapping] connector recycle failed: %v\n", err)
			continue
		}
		watchdog.mu.Lock()
		book.lastRecycle = now
		book.strikes = 0
		book.stallStrikes = 0
		book.severeStrikes = 0
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
