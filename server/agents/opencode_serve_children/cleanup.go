package opencode_serve_children

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

)

// CollectPIDs returns deduplicated PIDs from registry children and listeners on extraPorts.
func CollectPIDs(configHome string, extraPorts ...int) ([]int, error) {
	reg, err := Load(configHome)
	if err != nil {
		return nil, err
	}

	seen := make(map[int]struct{})
	var pids []int
	add := func(pid int) {
		if pid <= 0 {
			return
		}
		if _, ok := seen[pid]; ok {
			return
		}
		seen[pid] = struct{}{}
		pids = append(pids, pid)
	}

	for _, child := range reg.Children {
		add(child.PID)
	}

	for _, port := range extraPorts {
		for _, pid := range pidsOnPort(port) {
			add(pid)
		}
	}

	return pids, nil
}

// KillPIDs terminates verified opencode serve processes. Unverified PIDs are skipped.
func KillPIDs(configHome string, pids []int) (skipped []int, killed []int, err error) {
	reg, loadErr := Load(configHome)
	if loadErr != nil {
		return nil, nil, loadErr
	}
	portByPID := make(map[int]int, len(reg.Children))
	for _, child := range reg.Children {
		if child.PID > 0 && child.Port > 0 {
			portByPID[child.PID] = child.Port
		}
	}

	for _, pid := range pids {
		if pid <= 0 {
			continue
		}
		port := portByPID[pid]
		if !isOpencodeServeProcess(pid) {
			skipped = append(skipped, pid)
			continue
		}
		if port > 0 && !pidAmongOnPort(pid, port) {
			skipped = append(skipped, pid)
			continue
		}
		if killPID(pid) {
			killed = append(killed, pid)
		} else {
			skipped = append(skipped, pid)
		}
	}
	return skipped, killed, nil
}

// CleanupAll kills registered children, clears the children registry, and cleans
// related opencode serve registries (internal server, web server, auth-proxy backend).
func CleanupAll(configHome string, extraPorts ...int) error {
	pids, err := CollectPIDs(configHome, extraPorts...)
	if err != nil {
		return err
	}
	_, _, err = KillPIDs(configHome, pids)
	if err != nil {
		return err
	}

	if err := Clear(configHome); err != nil {
		return err
	}

	return cleanupRelatedRegistries(configHome, extraPorts...)
}

func cleanupRelatedRegistries(configHome string, extraPorts ...int) error {
	dataDir := ResolveDataDir(configHome)

	if err := killRegistryFile(filepath.Join(dataDir, "opencode-internal-server.json"), 0); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(dataDir, "opencode-internal-server.json"))

	for _, relPath := range []string{
		"procs/opencode-web/registry.json",
		"procs/basic-auth-proxy/registry.json",
		"procs/opencode-internal/registry.json",
	} {
		_ = killRegistryFile(filepath.Join(dataDir, relPath), 0)
		_ = os.Remove(filepath.Join(dataDir, relPath))
	}

	backendPort := readBackendPort(dataDir)
	ports := append([]int{}, extraPorts...)
	if backendPort > 0 {
		ports = append(ports, backendPort)
	}
	seen := make(map[int]struct{})
	for _, port := range ports {
		for _, pid := range pidsOnPort(port) {
			if _, ok := seen[pid]; ok {
				continue
			}
			seen[pid] = struct{}{}
			if isOpencodeServeProcess(pid) {
				_ = killPID(pid)
			}
		}
	}

	_ = os.Remove(filepath.Join(dataDir, "basic-auth-proxy.json"))
	return nil
}

func readBackendPort(dataDir string) int {
	data, err := os.ReadFile(filepath.Join(dataDir, "basic-auth-proxy.json"))
	if err != nil {
		return 0
	}
	var cfg struct {
		BackendPort int `json:"backend_port"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return 0
	}
	return cfg.BackendPort
}

func killRegistryFile(path string, expectedPort int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	var info struct {
		PID  int `json:"pid"`
		Port int `json:"port"`
	}
	if json.Unmarshal(data, &info) != nil || info.PID <= 0 {
		return nil
	}
	port := info.Port
	if expectedPort > 0 {
		port = expectedPort
	}
	if !isOpencodeServeProcess(info.PID) {
		return nil
	}
	if port > 0 && !pidAmongOnPort(info.PID, port) {
		return nil
	}
	_ = killPID(info.PID)
	return nil
}

func isOpencodeServeProcess(pid int) bool {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return false
	}
	cmd := strings.ToLower(strings.TrimSpace(string(out)))
	return strings.Contains(cmd, "opencode") && strings.Contains(cmd, "serve")
}

func pidsOnPort(port int) []int {
	if port <= 0 {
		return nil
	}
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port)).Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	var pids []int
	for _, field := range strings.Fields(strings.TrimSpace(string(out))) {
		pid, convErr := strconv.Atoi(field)
		if convErr != nil || pid <= 0 {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}

// pidAmongOnPort reports whether pid is among listeners on port (or port has no listeners).
func pidAmongOnPort(pid, port int) bool {
	if port <= 0 {
		return true
	}
	pids := pidsOnPort(port)
	if len(pids) == 0 {
		return true
	}
	for _, p := range pids {
		if p == pid {
			return true
		}
	}
	return false
}

func processRunning(pid int) bool {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "stat=").Output()
	if err != nil {
		return false
	}
	stat := strings.TrimSpace(string(out))
	if stat == "" {
		return false
	}
	return !strings.HasPrefix(stat, "Z")
}

func reapChild(pid int) {
	var status syscall.WaitStatus
	for i := 0; i < 20; i++ {
		wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
		if wpid > 0 || err != nil {
			return
		}
		if !processRunning(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func killPID(pid int) bool {
	if pid <= 0 {
		return false
	}
	if !isOpencodeServeProcess(pid) {
		return false
	}
	_ = syscall.Kill(pid, syscall.SIGTERM)
	time.Sleep(300 * time.Millisecond)
	if processRunning(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		time.Sleep(100 * time.Millisecond)
	}
	reapChild(pid)
	return !processRunning(pid)
}

// KillChild kills a verified opencode serve child and returns whether it was terminated.
func KillChild(pid, port int) bool {
	if pid <= 0 {
		return false
	}
	if !isOpencodeServeProcess(pid) {
		return false
	}
	if port > 0 && !pidAmongOnPort(pid, port) {
		return false
	}
	return killPID(pid)
}