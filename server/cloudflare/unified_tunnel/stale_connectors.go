package unified_tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// StaleConnector describes a cloudflared process that should not be running.
type StaleConnector struct {
	PID        int
	ConfigPath string
	TunnelRef  string
	Reason     string
}

// CloudflaredProcess captures a running cloudflared tunnel connector.
type CloudflaredProcess struct {
	PID        int
	Args       []string
	ConfigPath string
	TunnelRef  string
}

var listCloudflaredProcesses = defaultListCloudflaredProcesses

// ParseCloudflaredTunnelArgs extracts --config and the tunnel ref from a cloudflared argv slice.
func ParseCloudflaredTunnelArgs(args []string) (configPath, tunnelRef string, ok bool) {
	if len(args) == 0 {
		return "", "", false
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "cloudflared") || !strings.Contains(joined, "tunnel") {
		return "", "", false
	}

	runIdx := -1
	for i, arg := range args {
		if arg == "run" {
			runIdx = i
			break
		}
	}
	if runIdx < 0 || runIdx+1 >= len(args) {
		return "", "", false
	}
	tunnelRef = args[runIdx+1]

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--config" {
			configPath = args[i+1]
			break
		}
	}
	if configPath == "" {
		return "", "", false
	}
	return configPath, tunnelRef, true
}

func tunnelRefMatches(gotRef, wantRef, wantID string) bool {
	gotRef = strings.TrimSpace(gotRef)
	wantRef = strings.TrimSpace(wantRef)
	wantID = strings.TrimSpace(wantID)
	if gotRef == "" {
		return false
	}
	if wantRef != "" && strings.EqualFold(gotRef, wantRef) {
		return true
	}
	return wantID != "" && strings.EqualFold(gotRef, wantID)
}

func canonicalConfigPath(path string) string {
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// FindStaleTunnelConnectors returns cloudflared connectors for the tunnel that use
// a non-canonical or missing config file. keepPID is preserved (managed connector).
func FindStaleTunnelConnectors(tunnelRef, tunnelID, canonicalCfgPath string, keepPID int) ([]StaleConnector, error) {
	canonicalCfgPath = canonicalConfigPath(canonicalCfgPath)
	procs, err := listCloudflaredProcesses()
	if err != nil {
		return nil, err
	}

	var stale []StaleConnector
	for _, proc := range procs {
		if proc.PID == keepPID {
			continue
		}
		cfgPath, ref, ok := ParseCloudflaredTunnelArgs(proc.Args)
		if !ok {
			continue
		}
		if !tunnelRefMatches(ref, tunnelRef, tunnelID) {
			continue
		}

		reason := ""
		cfgAbs := canonicalConfigPath(cfgPath)
		if _, err := os.Stat(cfgPath); err != nil {
			if os.IsNotExist(err) {
				reason = fmt.Sprintf("missing config file %s", cfgPath)
			} else {
				reason = fmt.Sprintf("config file %s: %v", cfgPath, err)
			}
		} else if cfgAbs != canonicalCfgPath {
			reason = fmt.Sprintf("non-canonical config %s (want %s)", cfgPath, canonicalCfgPath)
		}
		if reason == "" {
			continue
		}
		stale = append(stale, StaleConnector{
			PID:        proc.PID,
			ConfigPath: cfgPath,
			TunnelRef:  ref,
			Reason:     reason,
		})
	}
	return stale, nil
}

// ReconcileStaleTunnelConnectors kills stale connectors for the given tunnel.
func ReconcileStaleTunnelConnectors(tunnelRef, tunnelID, canonicalCfgPath string, keepPID int) ([]int, error) {
	stale, err := FindStaleTunnelConnectors(tunnelRef, tunnelID, canonicalCfgPath, keepPID)
	if err != nil {
		return nil, err
	}
	var killed []int
	for _, conn := range stale {
		if err := killProcessPID(conn.PID); err != nil {
			return killed, fmt.Errorf("kill stale cloudflared pid %d: %w", conn.PID, err)
		}
		killed = append(killed, conn.PID)
	}
	return killed, nil
}

func killProcessPID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid %d", pid)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return proc.Kill()
	}
	return nil
}

func defaultListCloudflaredProcesses() ([]CloudflaredProcess, error) {
	out, err := exec.Command("pgrep", "-f", "cloudflared.*tunnel").Output()
	if err != nil {
		if len(out) == 0 {
			return nil, nil
		}
	}
	var procs []CloudflaredProcess
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil || pid <= 0 {
			continue
		}
		args, err := readProcessArgs(pid)
		if err != nil || len(args) == 0 {
			continue
		}
		cfg, ref, ok := ParseCloudflaredTunnelArgs(args)
		if !ok {
			continue
		}
		procs = append(procs, CloudflaredProcess{
			PID:        pid,
			Args:       args,
			ConfigPath: cfg,
			TunnelRef:  ref,
		})
	}
	return procs, nil
}

func readProcessArgs(pid int) ([]string, error) {
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err == nil {
		if len(data) == 0 {
			return nil, fmt.Errorf("empty cmdline for pid %d", pid)
		}
		parts := strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
		var args []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				args = append(args, part)
			}
		}
		return args, nil
	}

	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-ww", "-o", "args=").Output()
	if err != nil {
		return nil, err
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return nil, fmt.Errorf("empty ps output for pid %d", pid)
	}
	return strings.Fields(line), nil
}