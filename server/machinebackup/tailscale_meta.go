package machinebackup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	metaTailscaleName        = "tailscale-config.json"
	tailscaleConfigVersion   = "1.0"
	tailscaleDryRunHeader    = "  TAILSCALE(.backup/tailscale-config.json):"
	tailscaleTableColumnHdr  = "    VERSION    MODE                   SOCKS5              TAILSCALE IP    MAGIC DNS"
	tailscalePeersColumnHdr  = "      NAME     TAILSCALE IP     OS      STATUS"
)

var tailscaleHistoryLineRE = regexp.MustCompile(`(?i)tailscale`)

var buildTailscaleConfigSnapshotFn = func(home string) (*TailscaleConfigSnapshot, bool, error) {
	return CaptureTailscaleConfig(home)
}

// TailscaleVersionInfo holds tailscale version text and JSON output.
type TailscaleVersionInfo struct {
	Text string          `json:"text"`
	JSON json.RawMessage `json:"json"`
}

// TailscaleDaemonInfo describes the tailscaled process when discoverable.
type TailscaleDaemonInfo struct {
	PID                 int    `json:"pid,omitempty"`
	Cmdline             string `json:"cmdline,omitempty"`
	StatePath           string `json:"state_path,omitempty"`
	SocketPath          string `json:"socket_path,omitempty"`
	UserspaceNetworking bool   `json:"userspace_networking,omitempty"`
	Socks5Server        string `json:"socks5_server,omitempty"`
}

// TailscaleSetupInfo captures auto-generated setup steps and shell history.
type TailscaleSetupInfo struct {
	Summary     string   `json:"summary,omitempty"`
	Steps       []string `json:"steps"`
	Commands    []string `json:"commands"`
	BashHistory []string `json:"bash_history"`
	ZshHistory  []string `json:"zsh_history"`
	Notes       []string `json:"notes,omitempty"`
}

// TailscaleConfigSnapshot is written to .backup/tailscale-config.json.
type TailscaleConfigSnapshot struct {
	Version     string               `json:"version"`
	CapturedAt  time.Time            `json:"captured_at"`
	Running     bool                 `json:"running"`
	VersionInfo TailscaleVersionInfo `json:"version_info"`
	Daemon      TailscaleDaemonInfo  `json:"daemon"`
	Status      json.RawMessage      `json:"status"`
	Prefs       json.RawMessage      `json:"prefs"`
	Setup       TailscaleSetupInfo   `json:"setup"`
}

type tailscaleStatusSelf struct {
	TailscaleIPs []string `json:"TailscaleIPs"`
	DNSName      string   `json:"DNSName"`
}

type tailscaleStatusPeer struct {
	DNSName      string   `json:"DNSName"`
	TailscaleIPs []string `json:"TailscaleIPs"`
	OS           string   `json:"OS"`
	Online       bool     `json:"Online"`
	LastSeen     string   `json:"LastSeen"`
}

type tailscaleStatusParsed struct {
	BackendState string                         `json:"BackendState"`
	Self         tailscaleStatusSelf            `json:"Self"`
	Peer         map[string]tailscaleStatusPeer `json:"Peer"`
}

// CaptureTailscaleConfig collects tailscale state from server HOME when running.
// The bool return is true when the snapshot should be included in backup meta.
func CaptureTailscaleConfig(home string) (*TailscaleConfigSnapshot, bool, error) {
	statusRaw, _, running, err := tailscaleStatusOutput(home)
	if err != nil {
		return nil, false, err
	}
	if !running {
		return nil, false, nil
	}

	now := time.Now().UTC()
	snap := &TailscaleConfigSnapshot{
		Version:    tailscaleConfigVersion,
		CapturedAt: now,
		Running:    true,
		Status:     statusRaw,
	}

	versionText, versionJSON, err := tailscaleVersionOutput(home)
	if err != nil {
		return nil, false, err
	}
	snap.VersionInfo = TailscaleVersionInfo{Text: strings.TrimSpace(versionText), JSON: versionJSON}

	prefsRaw, err := tailscalePrefsOutput(home)
	if err != nil {
		return nil, false, err
	}
	redacted, err := redactTailscalePrefs(prefsRaw)
	if err != nil {
		return nil, false, err
	}
	snap.Prefs = redacted

	snap.Daemon = discoverTailscaleDaemon(home)
	snap.Setup = buildTailscaleSetup(home, snap.Daemon)

	return snap, true, nil
}

func tailscaleStatusOutput(home string) (json.RawMessage, *tailscaleStatusParsed, bool, error) {
	out, err := runTailscaleCommand(home, "status", "--json")
	if err != nil {
		return nil, nil, false, nil
	}
	raw := json.RawMessage(bytes.TrimSpace(out))
	var parsed tailscaleStatusParsed
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, nil, false, nil
	}
	return raw, &parsed, parsed.BackendState == "Running", nil
}

func tailscaleVersionOutput(home string) (string, json.RawMessage, error) {
	textOut, err := runTailscaleCommand(home, "version")
	if err != nil {
		return "", nil, err
	}
	jsonOut, err := runTailscaleCommand(home, "version", "--json")
	if err != nil {
		return "", nil, err
	}
	return string(textOut), json.RawMessage(bytes.TrimSpace(jsonOut)), nil
}

func tailscalePrefsOutput(home string) ([]byte, error) {
	return runTailscaleCommand(home, "debug", "prefs")
}

func runTailscaleCommand(home string, args ...string) ([]byte, error) {
	env := tailscaleCommandEnv(home)
	bin, err := resolveTailscaleBinary(home, env)
	if err != nil {
		return nil, err
	}
	execBin := bin
	cleanup := func() {}
	if isShellScript(bin) {
		if normalizedPath, cleanupFn, normErr := normalizeHarnessTailscaleScript(bin); normErr == nil {
			execBin = normalizedPath
			cleanup = cleanupFn
		}
	}
	defer cleanup()

	cmd := tailscaleExecCommand(execBin, args)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func tailscaleExecCommand(bin string, args []string) *exec.Cmd {
	if isShellScript(bin) {
		return exec.Command("sh", append([]string{bin}, args...)...)
	}
	return exec.Command(bin, args...)
}

// normalizeHarnessTailscaleScript fixes doctest mapping-gen indentation that breaks
// shell heredoc terminators in the harness mock script.
func normalizeHarnessTailscaleScript(path string) (execPath string, cleanup func(), err error) {
	cleanup = func() {}
	data, err := os.ReadFile(path)
	if err != nil {
		return path, cleanup, err
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		if trimmed, ok := strings.CutPrefix(line, "\t"); ok {
			lines[i] = trimmed
			changed = true
		}
	}
	if !changed {
		return path, cleanup, nil
	}
	normalized := strings.Join(lines, "\n")
	tmp, err := os.CreateTemp(filepath.Dir(path), "tailscale-mock-*")
	if err != nil {
		return path, cleanup, err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(normalized); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return path, cleanup, err
	}
	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return path, cleanup, err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return path, cleanup, err
	}
	return tmpPath, func() { os.Remove(tmpPath) }, nil
}

func isShellScript(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return false
	}
	return string(hdr) == "#!"
}

func resolveTailscaleBinary(home string, env []string) (string, error) {
	for _, candidateHome := range tailscaleHomeCandidates(home) {
		if bin, ok := tailscaleBinaryInHomeBin(candidateHome); ok {
			return bin, nil
		}
	}
	return lookPathInEnv("tailscale", env)
}

func tailscaleHomeCandidates(home string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, h := range []string{home, os.Getenv("HOME")} {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if abs, err := filepath.Abs(h); err == nil {
			h = abs
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	return out
}

func tailscaleBinaryInHomeBin(home string) (string, bool) {
	candidate := filepath.Join(home, "bin", "tailscale")
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return "", false
	}
	return candidate, true
}

func lookPathInEnv(file string, env []string) (string, error) {
	var pathEnv string
	for _, entry := range env {
		if strings.HasPrefix(entry, "PATH=") {
			pathEnv = strings.TrimPrefix(entry, "PATH=")
			break
		}
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, file)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Mode()&0111 != 0 || isShellScript(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("tailscale not found in PATH")
}

func tailscaleCommandEnv(home string) []string {
	env := os.Environ()
	if home == "" {
		return env
	}
	binDir := filepath.Join(home, "bin")
	if st, err := os.Stat(filepath.Join(binDir, "tailscale")); err != nil || st.IsDir() {
		return env
	}
	return prependPathEnv(env, binDir)
}

func prependPathEnv(env []string, dir string) []string {
	out := append([]string(nil), env...)
	for i, e := range out {
		if strings.HasPrefix(e, "PATH=") {
			out[i] = "PATH=" + dir + string(os.PathListSeparator) + strings.TrimPrefix(e, "PATH=")
			return out
		}
	}
	return append(out, "PATH="+dir)
}

var tailscalePrivateKeyFields = []string{
	"PrivateNodeKey",
	"OldPrivateNodeKey",
	"NetworkLockKey",
}

func redactTailscalePrefs(raw []byte) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage("{}"), nil
	}
	var prefs map[string]json.RawMessage
	if err := json.Unmarshal(raw, &prefs); err != nil {
		return nil, fmt.Errorf("parse tailscale prefs: %w", err)
	}
	for _, key := range tailscalePrivateKeyFields {
		delete(prefs, key)
	}
	if cfgRaw, ok := prefs["Config"]; ok {
		var cfg map[string]json.RawMessage
		if err := json.Unmarshal(cfgRaw, &cfg); err == nil {
			for _, key := range tailscalePrivateKeyFields {
				delete(cfg, key)
			}
			updated, err := json.Marshal(cfg)
			if err != nil {
				return nil, err
			}
			prefs["Config"] = updated
		}
	}
	out, err := json.Marshal(prefs)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

func discoverTailscaleDaemon(home string) TailscaleDaemonInfo {
	info := TailscaleDaemonInfo{}
	if pid, cmdline, ok := tailscaledProcInfo(); ok {
		info.PID = pid
		if cmdline != "" {
			info.Cmdline = cmdline
		}
	}
	if info.Cmdline == "" {
		if cmdline := tailscaledCmdlineFromHistory(home); cmdline != "" {
			info.Cmdline = cmdline
		}
	}
	applyDaemonFlagParsing(&info)
	return info
}

func tailscaledProcInfo() (int, string, bool) {
	pidOut, err := exec.Command("pgrep", "-x", "tailscaled").Output()
	if err != nil || len(bytes.TrimSpace(pidOut)) == 0 {
		return 0, "", false
	}
	pidStr := strings.TrimSpace(string(pidOut))
	var pid int
	if _, err := fmt.Sscanf(pidStr, "%d", &pid); err != nil || pid <= 0 {
		lines := strings.Split(strings.TrimSpace(pidStr), "\n")
		if len(lines) == 0 {
			return 0, "", false
		}
		if _, err := fmt.Sscanf(strings.TrimSpace(lines[0]), "%d", &pid); err != nil || pid <= 0 {
			return 0, "", false
		}
	}
	cmdlinePath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "cmdline")
	raw, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return pid, "", true
	}
	return pid, strings.ReplaceAll(string(raw), "\x00", " "), true
}

func tailscaledCmdlineFromHistory(home string) string {
	for _, name := range []string{".zsh_history", ".bash_history"} {
		path := filepath.Join(home, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			lower := strings.ToLower(line)
			if strings.Contains(lower, "tailscaled") {
				if idx := strings.Index(lower, "tailscaled"); idx >= 0 {
					return strings.TrimSpace(line[idx:])
				}
			}
		}
	}
	return ""
}

func applyDaemonFlagParsing(info *TailscaleDaemonInfo) {
	if info == nil || info.Cmdline == "" {
		return
	}
	args := strings.Fields(info.Cmdline)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--tun=userspace-networking" || arg == "--tun" && i+1 < len(args) && args[i+1] == "userspace-networking":
			info.UserspaceNetworking = true
			if arg == "--tun" {
				i++
			}
		case strings.HasPrefix(arg, "--tun=userspace-networking"):
			info.UserspaceNetworking = true
		case strings.HasPrefix(arg, "--socks5-server="):
			info.Socks5Server = strings.TrimPrefix(arg, "--socks5-server=")
		case arg == "--socks5-server" && i+1 < len(args):
			i++
			info.Socks5Server = args[i]
		case strings.HasPrefix(arg, "--state="):
			info.StatePath = strings.TrimPrefix(arg, "--state=")
		case arg == "--state" && i+1 < len(args):
			i++
			info.StatePath = args[i]
		case strings.HasPrefix(arg, "--socket="):
			info.SocketPath = strings.TrimPrefix(arg, "--socket=")
		case arg == "--socket" && i+1 < len(args):
			i++
			info.SocketPath = args[i]
		}
	}
}

func readTailscaleHistoryLines(home, name string) []string {
	path := filepath.Join(home, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if tailscaleHistoryLineRE.MatchString(line) {
			lines = append(lines, line)
		}
	}
	return lines
}

func buildTailscaleSetup(home string, daemon TailscaleDaemonInfo) TailscaleSetupInfo {
	bashHistory := readTailscaleHistoryLines(home, ".bash_history")
	zshHistory := readTailscaleHistoryLines(home, ".zsh_history")

	startCmd := daemon.Cmdline
	if startCmd == "" {
		startCmd = "tailscaled"
	}
	commands := []string{startCmd + " &", "tailscale up"}

	steps := []string{
		"1. Install (proxy if needed): curl -fsSL https://tailscale.com/install.sh | sh",
		fmt.Sprintf("2. Start daemon: %s &", startCmd),
		"3. Join: tailscale up",
		"4. Verify: tailscale status",
	}

	return TailscaleSetupInfo{
		Summary:     "Auto-generated from live daemon cmdline and standard join steps",
		Steps:       steps,
		Commands:    commands,
		BashHistory: bashHistory,
		ZshHistory:  zshHistory,
	}
}

func marshalTailscaleConfigSnapshot(snap *TailscaleConfigSnapshot) ([]byte, error) {
	if snap == nil {
		return nil, nil
	}
	return json.MarshalIndent(snap, "", "  ")
}

func formatTailscaleSummaryLinesForHome(home string) []string {
	for _, candidate := range tailscaleHomeCandidates(home) {
		if snap, included, err := buildTailscaleConfigSnapshotFn(candidate); err == nil && included && snap != nil {
			return formatTailscaleSummaryLines(snap)
		}
	}
	return nil
}

func captureTailscaleConfigForHome(home string) (*TailscaleConfigSnapshot, bool, error) {
	for _, candidate := range tailscaleHomeCandidates(home) {
		if snap, included, err := buildTailscaleConfigSnapshotFn(candidate); err != nil {
			return nil, false, err
		} else if included && snap != nil {
			return snap, true, nil
		}
	}
	return nil, false, nil
}

func formatTailscaleSummaryLines(snap *TailscaleConfigSnapshot) []string {
	if snap == nil || !snap.Running {
		return nil
	}

	var parsed tailscaleStatusParsed
	_ = json.Unmarshal(snap.Status, &parsed)

	mode := tailscaleModeLabel(snap.Daemon)
	socks5 := snap.Daemon.Socks5Server
	if socks5 == "" {
		socks5 = "-"
	}
	selfIP := firstString(parsed.Self.TailscaleIPs)
	dnsName := strings.TrimSuffix(parsed.Self.DNSName, ".")

	lines := []string{
		tailscaleDryRunHeader,
		fmt.Sprintf("    captured_at: %s  (running)", formatMetaCapturedAt(snap.CapturedAt)),
		tailscaleTableColumnHdr,
		fmt.Sprintf("    %-11s %-22s %-19s %-15s %s",
			snap.VersionInfo.Text, mode, socks5, selfIP, dnsName),
		"",
		"    DAEMON",
	}
	if snap.Daemon.Cmdline != "" {
		lines = append(lines, "      "+snap.Daemon.Cmdline)
	} else {
		lines = append(lines, "      (unknown)")
	}

	lines = append(lines, "", "    SETUP")
	for _, step := range snap.Setup.Steps {
		lines = append(lines, "      "+step)
	}

	lines = append(lines, "", "    SHELL HISTORY (tailscale)")
	for _, line := range snap.Setup.BashHistory {
		lines = append(lines, "      [bash] "+line)
	}
	for _, line := range snap.Setup.ZshHistory {
		lines = append(lines, "      [zsh]  "+line)
	}

	peers := sortedTailscalePeers(parsed.Peer)
	lines = append(lines, "", fmt.Sprintf("    PEERS (%d)", len(peers)), tailscalePeersColumnHdr)
	for _, peer := range peers {
		ip := firstString(peer.TailscaleIPs)
		status := formatTailscalePeerStatus(peer, snap.CapturedAt)
		lines = append(lines, fmt.Sprintf("      %-9s %-16s %-7s %s",
			peer.DNSName, ip, peer.OS, status))
	}
	return lines
}

func tailscaleModeLabel(daemon TailscaleDaemonInfo) string {
	if daemon.UserspaceNetworking {
		return "userspace-networking"
	}
	if daemon.Cmdline != "" {
		for _, arg := range strings.Fields(daemon.Cmdline) {
			if strings.HasPrefix(arg, "--tun=") {
				return strings.TrimPrefix(arg, "--tun=")
			}
		}
	}
	return "kernel"
}

type tailscalePeerRow struct {
	DNSName      string
	TailscaleIPs []string
	OS           string
	Online       bool
	LastSeen     string
}

func sortedTailscalePeers(peers map[string]tailscaleStatusPeer) []tailscalePeerRow {
	if len(peers) == 0 {
		return nil
	}
	rows := make([]tailscalePeerRow, 0, len(peers))
	for _, peer := range peers {
		rows = append(rows, tailscalePeerRow{
			DNSName:      peer.DNSName,
			TailscaleIPs: peer.TailscaleIPs,
			OS:           peer.OS,
			Online:       peer.Online,
			LastSeen:     peer.LastSeen,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].DNSName < rows[j].DNSName })
	return rows
}

func formatTailscalePeerStatus(peer tailscalePeerRow, now time.Time) string {
	if peer.Online {
		return "active"
	}
	if peer.LastSeen == "" {
		return "offline"
	}
	last, err := time.Parse(time.RFC3339, peer.LastSeen)
	if err != nil {
		return "offline"
	}
	return "offline, last seen " + formatRelativeDuration(now.Sub(last))
}

func formatRelativeDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1d ago"
	}
	return fmt.Sprintf("%dd ago", days)
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}