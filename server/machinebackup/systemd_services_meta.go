package machinebackup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	metaSystemdServicesName     = "systemd-services.json"
	systemdServicesVersion      = "1.0"
	systemdDryRunHeader         = "  SYSTEMD SERVICES(.backup/systemd-services.json):"
	systemdTableColumnHeader    = "      UNIT                     PID     DESCRIPTION"
)

var buildSystemdServicesSnapshotFn = func(home string) (*SystemdServicesSnapshot, bool, error) {
	return CaptureSystemdServices(home)
}

// SystemdUnitSnapshot is one running service unit in a scope.
type SystemdUnitSnapshot struct {
	Unit        string `json:"unit"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
	MainPID     int    `json:"main_pid"`
	UnitFile    string `json:"unit_file"`
}

// SystemdScopeSnapshot is the capture result for user or system scope.
type SystemdScopeSnapshot struct {
	Available    bool                  `json:"available"`
	RunningCount int                   `json:"running_count"`
	Error        string                `json:"error,omitempty"`
	Units        []SystemdUnitSnapshot `json:"units"`
}

// SystemdServicesScopes holds per-scope systemd capture results.
type SystemdServicesScopes struct {
	User   SystemdScopeSnapshot `json:"user"`
	System SystemdScopeSnapshot `json:"system"`
}

// SystemdServicesSnapshot is written to .backup/systemd-services.json.
type SystemdServicesSnapshot struct {
	Version          string                `json:"version"`
	CapturedAt       time.Time             `json:"captured_at"`
	SystemdAvailable bool                  `json:"systemd_available"`
	Scopes           SystemdServicesScopes `json:"scopes"`
}

type systemdListUnitEntry struct {
	Unit        string `json:"unit"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
}

// CaptureSystemdServices collects running systemd service units when systemctl is available.
// The bool return is true when the snapshot should be included in backup meta.
func CaptureSystemdServices(home string) (*SystemdServicesSnapshot, bool, error) {
	if !systemctlGatePassed(home) {
		return nil, false, nil
	}

	now := time.Now().UTC()
	snap := &SystemdServicesSnapshot{
		Version:          systemdServicesVersion,
		CapturedAt:       now,
		SystemdAvailable: true,
		Scopes: SystemdServicesScopes{
			User:   captureSystemdScope(home, true),
			System: captureSystemdScope(home, false),
		},
	}
	return snap, true, nil
}

func systemctlGatePassed(home string) bool {
	_, err := runSystemctlCommand(home, "--version")
	return err == nil
}

func captureSystemdScope(home string, userScope bool) SystemdScopeSnapshot {
	args := []string{"list-units", "--type=service", "--state=running", "--output=json"}
	if userScope {
		args = append([]string{"--user"}, args...)
	}

	out, err := runSystemctlCommand(home, args...)
	if err != nil {
		return SystemdScopeSnapshot{
			Available:    false,
			RunningCount: 0,
			Error:        formatSystemctlScopeError(err, out),
			Units:        []SystemdUnitSnapshot{},
		}
	}

	entries, err := parseSystemdListUnitsJSON(out)
	if err != nil {
		return SystemdScopeSnapshot{
			Available:    false,
			RunningCount: 0,
			Error:        err.Error(),
			Units:        []SystemdUnitSnapshot{},
		}
	}

	units := make([]SystemdUnitSnapshot, 0, len(entries))
	for _, entry := range entries {
		unit := SystemdUnitSnapshot{
			Unit:        entry.Unit,
			Load:        entry.Load,
			Active:      entry.Active,
			Sub:         entry.Sub,
			Description: entry.Description,
		}
		enrichSystemdUnitFromShow(home, userScope, &unit)
		units = append(units, unit)
	}
	sort.Slice(units, func(i, j int) bool { return units[i].Unit < units[j].Unit })

	return SystemdScopeSnapshot{
		Available:    true,
		RunningCount: len(units),
		Units:        units,
	}
}

func parseSystemdListUnitsJSON(out []byte) ([]systemdListUnitEntry, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 || string(trimmed) == "[]" {
		return nil, nil
	}
	var entries []systemdListUnitEntry
	if err := json.Unmarshal(trimmed, &entries); err != nil {
		return nil, fmt.Errorf("parse systemctl list-units json: %w", err)
	}
	return entries, nil
}

func enrichSystemdUnitFromShow(home string, userScope bool, unit *SystemdUnitSnapshot) {
	if unit == nil || unit.Unit == "" {
		return
	}
	args := []string{"show", unit.Unit, "--property=Description,MainPID,FragmentPath"}
	if userScope {
		args = append([]string{"--user"}, args...)
	}
	out, err := runSystemctlCommand(home, args...)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "Description":
			if value != "" {
				unit.Description = value
			}
		case "MainPID":
			if pid, err := strconv.Atoi(value); err == nil && pid > 0 {
				unit.MainPID = pid
			}
		case "FragmentPath":
			if value != "" {
				unit.UnitFile = value
			}
		}
	}
}

func runSystemctlCommand(home string, args ...string) ([]byte, error) {
	env := systemctlCommandEnv(home)
	bin, err := resolveSystemctlBinary(home, env)
	if err != nil {
		return nil, err
	}
	execBin := bin
	cleanup := func() {}
	if isShellScript(bin) {
		// mapping-gen indents harness mock scripts with tabs; normalize like tailscale mock.
		if normalizedPath, cleanupFn, normErr := normalizeHarnessTailscaleScript(bin); normErr == nil {
			execBin = normalizedPath
			cleanup = cleanupFn
		}
	}
	defer cleanup()

	cmd := systemctlExecCommand(execBin, args)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}
	return out, nil
}

func systemctlExecCommand(bin string, args []string) *exec.Cmd {
	if isShellScript(bin) {
		return exec.Command("sh", append([]string{bin}, args...)...)
	}
	return exec.Command(bin, args...)
}

func resolveSystemctlBinary(home string, env []string) (string, error) {
	for _, candidateHome := range tailscaleHomeCandidates(home) {
		if bin, ok := systemctlBinaryInHomeBin(candidateHome); ok {
			return bin, nil
		}
	}
	return lookPathInEnv("systemctl", env)
}

func systemctlBinaryInHomeBin(home string) (string, bool) {
	candidate := filepath.Join(home, "bin", "systemctl")
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return "", false
	}
	return candidate, true
}

func systemctlCommandEnv(home string) []string {
	env := os.Environ()
	if home == "" {
		return env
	}
	binDir := filepath.Join(home, "bin")
	if st, err := os.Stat(filepath.Join(binDir, "systemctl")); err != nil || st.IsDir() {
		return env
	}
	return prependPathEnv(env, binDir)
}

func formatSystemctlScopeError(err error, out []byte) string {
	msg := strings.TrimSpace(string(out))
	if msg != "" {
		if err != nil {
			return fmt.Sprintf("%v: %s", err, msg)
		}
		return msg
	}
	if err != nil {
		return err.Error()
	}
	return "unknown error"
}

func marshalSystemdServicesSnapshot(snap *SystemdServicesSnapshot) ([]byte, error) {
	if snap == nil {
		return nil, nil
	}
	return json.MarshalIndent(snap, "", "  ")
}

func formatSystemdServicesSummaryLinesForHome(home string) []string {
	for _, candidate := range tailscaleHomeCandidates(home) {
		if snap, included, err := buildSystemdServicesSnapshotFn(candidate); err == nil && included && snap != nil {
			return formatSystemdServicesSummaryLines(snap)
		}
	}
	return nil
}

func captureSystemdServicesForHome(home string) (*SystemdServicesSnapshot, bool, error) {
	for _, candidate := range tailscaleHomeCandidates(home) {
		if snap, included, err := buildSystemdServicesSnapshotFn(candidate); err != nil {
			return nil, false, err
		} else if included && snap != nil {
			return snap, true, nil
		}
	}
	return nil, false, nil
}

func formatSystemdServicesSummaryLines(snap *SystemdServicesSnapshot) []string {
	if snap == nil || !snap.SystemdAvailable {
		return nil
	}

	lines := []string{
		systemdDryRunHeader,
		fmt.Sprintf("    captured_at: %s  %s", formatMetaCapturedAt(snap.CapturedAt), formatSystemdRunningSubheader(snap)),
	}

	user := snap.Scopes.User
	system := snap.Scopes.System

	if user.Available && user.RunningCount > 0 {
		lines = append(lines, "", fmt.Sprintf("    USER (%d)", user.RunningCount), systemdTableColumnHeader)
		for _, unit := range user.Units {
			lines = append(lines, formatSystemdUnitRow(unit))
		}
	} else if !user.Available {
		lines = append(lines, "", "    USER", fmt.Sprintf("      (unavailable: %s)", formatSystemdDisplayError(user.Error)))
	}

	if system.Available && system.RunningCount > 0 {
		lines = append(lines, "", fmt.Sprintf("    SYSTEM (%d)", system.RunningCount), systemdTableColumnHeader)
		for _, unit := range system.Units {
			lines = append(lines, formatSystemdUnitRow(unit))
		}
	} else if !system.Available {
		lines = append(lines, "", "    SYSTEM", fmt.Sprintf("      (unavailable: %s)", formatSystemdDisplayError(system.Error)))
	}

	return lines
}

func formatSystemdUnitRow(unit SystemdUnitSnapshot) string {
	return fmt.Sprintf("      %-25s %-7d %s", unit.Unit, unit.MainPID, unit.Description)
}

func formatSystemdRunningSubheader(snap *SystemdServicesSnapshot) string {
	user := snap.Scopes.User
	system := snap.Scopes.System

	total := 0
	if user.Available {
		total += user.RunningCount
	}
	if system.Available {
		total += system.RunningCount
	}

	if total == 0 && user.Available && system.Available {
		return "(0 running)"
	}

	var availParts []string
	if user.Available {
		availParts = append(availParts, fmt.Sprintf("%d user", user.RunningCount))
	}
	if system.Available {
		availParts = append(availParts, fmt.Sprintf("%d system", system.RunningCount))
	}

	var unavailParts []string
	if !user.Available {
		unavailParts = append(unavailParts, "user unavailable")
	}
	if !system.Available {
		unavailParts = append(unavailParts, "system unavailable")
	}

	suffix := strings.Join(availParts, ", ")
	if len(unavailParts) > 0 {
		if suffix != "" {
			suffix += "; " + strings.Join(unavailParts, "; ")
		} else {
			suffix = strings.Join(unavailParts, "; ")
		}
	}
	if suffix == "" {
		return "(0 running)"
	}
	return fmt.Sprintf("(%d running: %s)", total, suffix)
}

func formatSystemdDisplayError(errMsg string) string {
	errMsg = strings.TrimSpace(errMsg)
	if strings.HasPrefix(errMsg, "exit status ") {
		if idx := strings.Index(errMsg, ": "); idx >= 0 {
			return errMsg[idx+2:]
		}
	}
	return errMsg
}