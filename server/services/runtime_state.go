package services

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Runtime records let a fresh server instance identify processes left behind by
// a previous one. Service PIDs live only in memory, so after a crash (or a
// shutdown that was cut short) nothing knows the orphan exists: it keeps its
// port, and the new instance's copy dies with "address already in use".
//
// The file is runtime state, not user configuration, so it lives beside
// services.json rather than inside it.
const runtimeStateVersion = 1

type serviceRuntimeRecord struct {
	PID       int    `json:"pid"`
	PGID      int    `json:"pgid,omitempty"`
	StartedAt string `json:"startedAt,omitempty"`
	// Command is what the process was launched to run. Reconciliation only kills
	// a pid whose live command line still matches, so a recycled pid cannot be
	// mistaken for our orphan.
	Command string `json:"command,omitempty"`
}

type serviceRuntimeState struct {
	Version   int                             `json:"version"`
	Processes map[string]serviceRuntimeRecord `json:"processes"`
}

func (m *Manager) runtimeStateFile() string {
	return filepath.Join(m.servicesDataDir(), "services-runtime.json")
}

// recordRuntimeProcess persists the pid a service just started under.
func (m *Manager) recordRuntimeProcess(id string, pid int, pgid int, command string) {
	state, err := m.loadRuntimeState()
	if err != nil {
		return
	}
	if state.Processes == nil {
		state.Processes = map[string]serviceRuntimeRecord{}
	}
	state.Processes[id] = serviceRuntimeRecord{
		PID:       pid,
		PGID:      pgid,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Command:   command,
	}
	m.saveRuntimeState(state)
}

// forgetRuntimeProcess drops a service's record once its process is gone.
func (m *Manager) forgetRuntimeProcess(id string) {
	state, err := m.loadRuntimeState()
	if err != nil || state.Processes == nil {
		return
	}
	if _, ok := state.Processes[id]; !ok {
		return
	}
	delete(state.Processes, id)
	m.saveRuntimeState(state)
}

func (m *Manager) loadRuntimeState() (serviceRuntimeState, error) {
	state := serviceRuntimeState{Version: runtimeStateVersion}
	data, err := os.ReadFile(m.runtimeStateFile())
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		// A corrupt record must not block startup; reconciliation just has
		// nothing to work with.
		return serviceRuntimeState{Version: runtimeStateVersion}, nil
	}
	if state.Processes == nil {
		state.Processes = map[string]serviceRuntimeRecord{}
	}
	return state, nil
}

func (m *Manager) saveRuntimeState(state serviceRuntimeState) {
	state.Version = runtimeStateVersion
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	path := m.runtimeStateFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	// Write-then-rename so a crash mid-write cannot leave a truncated record.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// ReconcileOrphans kills processes recorded by a previous instance that are
// still running, returning the names it reclaimed. Call it before autostart so
// reclaimed ports are free for the new processes.
func ReconcileOrphans() []string {
	return defaultManager.ReconcileOrphans()
}

func (m *Manager) ReconcileOrphans() []string {
	state, err := m.loadRuntimeState()
	if err != nil || len(state.Processes) == 0 {
		return nil
	}

	var reclaimed []string
	for id, record := range state.Processes {
		if record.PID <= 0 || !processAlive(record.PID) {
			delete(state.Processes, id)
			continue
		}
		if !runtimeCommandMatches(record) {
			// The pid was recycled by an unrelated process; leave it alone.
			fmt.Printf("[services] not reclaiming %s: pid %d no longer runs the recorded command\n", id, record.PID)
			delete(state.Processes, id)
			continue
		}
		name := m.serviceName(id)
		fmt.Printf("[services] reclaiming orphaned process for %q (pid %d) left by a previous instance\n", name, record.PID)
		killProcessGroupNow(record.PID)
		m.clearProcessPIDByPID(record.PID)
		delete(state.Processes, id)
		reclaimed = append(reclaimed, name)
	}
	m.saveRuntimeState(state)
	return reclaimed
}

// serviceName resolves a display name for logging, falling back to the id.
func (m *Manager) serviceName(id string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if def, ok := m.findDefinitionLocked(id); ok && def.Name != "" {
		return def.Name
	}
	return id
}

// runtimeCommandMatches reports whether the live process still looks like the one
// the record describes.
func runtimeCommandMatches(record serviceRuntimeRecord) bool {
	if strings.TrimSpace(record.Command) == "" {
		return false
	}
	live, err := processCommandLine(record.PID)
	if err != nil || strings.TrimSpace(live) == "" {
		return false
	}
	return commandMatches(record.Command, live)
}

// commandMatches compares a recorded service command with a live command line.
// The service runs under `bash -lc`, so the live line is the shell's; requiring
// a substring match of the recorded command keeps this robust across shells
// while still rejecting an unrelated pid.
func commandMatches(recorded, live string) bool {
	recorded = strings.TrimSpace(recorded)
	live = strings.TrimSpace(live)
	if recorded == "" || live == "" {
		return false
	}
	return strings.Contains(live, recorded)
}

// processCommandLine returns a process's command line for identity checks.
// Linux reads /proc directly; other platforms fall back to ps so the same guard
// works on a developer machine.
func processCommandLine(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid pid %d", pid)
	}
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			return "", err
		}
		// cmdline is NUL-separated with a trailing NUL.
		parts := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
		return strings.Join(parts, " "), nil
	}
	out, err := exec.Command("ps", "-o", "command=", "-p", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// killProcessGroupNow SIGKILLs a process group, escalating immediately because
// the process already had its graceful window.
func killProcessGroupNow(pid int) {
	pgid, err := processGroupID(pid)
	if err != nil {
		pgid = pid
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && processAlive(pid) {
		time.Sleep(50 * time.Millisecond)
	}
}
