package portforward

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	pkgcf "github.com/xhd2015/dot-pkgs/go-pkgs/cloudflare"

	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
	"github.com/xhd2015/ai-critic/server/domains"
)

var reconcileOwnedConnectors = defaultReconcileOwnedConnectors

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// adoptKeepPID returns the connector PID to keep. If the recorded PID is dead,
// adopt the newest live canonical PID instead of SIGTERM-ing it as a duplicate.
func adoptKeepPID(keepPID int, live []int, alive func(int) bool) int {
	if keepPID > 0 && alive(keepPID) {
		return keepPID
	}
	if len(live) == 0 {
		return 0
	}
	keep := live[0]
	for _, pid := range live[1:] {
		if pid > keep {
			keep = pid
		}
	}
	return keep
}

func defaultReconcileOwnedConnectors() {
	cfg, err := domains.LoadDomains()
	if err != nil || cfg == nil {
		return
	}
	name := strings.TrimSpace(cfg.TunnelName)
	if name == "" {
		return
	}
	configDir, err := pkgcf.DefaultConfigDir()
	if err != nil {
		fmt.Printf("[port mapping] owned tunnel config dir: %v\n", err)
		return
	}
	dir, err := pkgcf.ManagedTunnelDir(configDir, name)
	if err != nil {
		return
	}
	st, err := pkgcf.LoadTunnelState(dir)
	if err != nil || st == nil {
		return
	}
	cfgPath := filepath.Join(dir, "config.yml")
	live, err := unified_tunnel.CanonicalPIDs(st.TunnelName, st.TunnelID, cfgPath)
	if err != nil {
		fmt.Printf("[port mapping] list connectors: %v\n", err)
		return
	}
	keepPID := adoptKeepPID(st.ConnectorPID, live, pidAlive)
	if keepPID != st.ConnectorPID {
		st.ConnectorPID = keepPID
		if saveErr := pkgcf.SaveTunnelState(dir, st); saveErr != nil {
			fmt.Printf("[port mapping] adopt connector pid: %v\n", saveErr)
		} else if keepPID > 0 {
			fmt.Printf("[port mapping] adopted live cloudflared pid %d\n", keepPID)
		}
	}
	killed, err := unified_tunnel.ReconcileStaleTunnelConnectors(st.TunnelName, st.TunnelID, cfgPath, keepPID)
	if err != nil {
		fmt.Printf("[port mapping] reconcile connectors: %v\n", err)
		return
	}
	if len(killed) > 0 {
		fmt.Printf("[port mapping] killed duplicate cloudflared PIDs: %v\n", killed)
	}
}
