package machinebackup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	metaCloudflaredName       = "cloudflared-config.json"
	cloudflaredConfigVersion  = "1.0"
	cloudflaredDryRunHeader   = "  CLOUDFLARED(.backup/cloudflared-config.json):"
	cloudflaredTableColumnHdr = "    VERSION              MODE           TARGET"
	cloudflaredTunnelListErr  = "tunnel list requires cloudflare credentials"
)

var cloudflaredHistoryLineRE = regexp.MustCompile(`(?i)cloudflared`)

var buildCloudflaredConfigSnapshotFn = func(home string) (*CloudflaredConfigSnapshot, bool, error) {
	return CaptureCloudflaredConfig(home)
}

// CloudflaredVersionInfo holds cloudflared version text and optional JSON output.
type CloudflaredVersionInfo struct {
	Text string          `json:"text"`
	JSON json.RawMessage `json:"json,omitempty"`
}

// CloudflaredProcessInfo describes the running cloudflared process when discoverable.
type CloudflaredProcessInfo struct {
	PID     int    `json:"pid,omitempty"`
	Cmdline string `json:"cmdline,omitempty"`
}

// CloudflaredQuickTunnelInfo captures quick-tunnel flags from the running cmdline.
type CloudflaredQuickTunnelInfo struct {
	URL      string `json:"url,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

// CloudflaredTunnelsInfo is the best-effort tunnel list capture result.
type CloudflaredTunnelsInfo struct {
	Available bool              `json:"available"`
	Error     string            `json:"error,omitempty"`
	Items     []json.RawMessage `json:"items"`
}

// CloudflaredConfigFileInfo describes ~/.cloudflared/config.yml when present.
type CloudflaredConfigFileInfo struct {
	Path         string `json:"path"`
	Present      bool   `json:"present"`
	RedactedYAML string `json:"redacted_yaml,omitempty"`
}

// CloudflaredSetupInfo captures shell history lines mentioning cloudflared.
type CloudflaredSetupInfo struct {
	BashHistory []string `json:"bash_history"`
	ZshHistory  []string `json:"zsh_history"`
}

// CloudflaredConfigSnapshot is written to .backup/cloudflared-config.json.
type CloudflaredConfigSnapshot struct {
	Version     string                     `json:"version"`
	CapturedAt  time.Time                  `json:"captured_at"`
	Running     bool                       `json:"running"`
	VersionInfo CloudflaredVersionInfo     `json:"version_info"`
	Process     CloudflaredProcessInfo     `json:"process"`
	QuickTunnel CloudflaredQuickTunnelInfo `json:"quick_tunnel"`
	Tunnels     CloudflaredTunnelsInfo       `json:"tunnels"`
	Config      CloudflaredConfigFileInfo    `json:"config"`
	Setup       CloudflaredSetupInfo         `json:"setup"`
}

// CaptureCloudflaredConfig collects cloudflared state from server HOME when running.
// The bool return is true when the snapshot should be included in backup meta.
func CaptureCloudflaredConfig(home string) (*CloudflaredConfigSnapshot, bool, error) {
	if !cloudflaredRunningGate(home) {
		return nil, false, nil
	}

	now := time.Now().UTC()
	snap := &CloudflaredConfigSnapshot{
		Version:    cloudflaredConfigVersion,
		CapturedAt: now,
		Running:    true,
		Tunnels: CloudflaredTunnelsInfo{
			Items: []json.RawMessage{},
		},
	}

	versionText, versionJSON, err := cloudflaredVersionOutput(home)
	if err != nil {
		return nil, false, err
	}
	snap.VersionInfo = CloudflaredVersionInfo{Text: strings.TrimSpace(versionText), JSON: versionJSON}

	snap.Process = discoverCloudflaredProcess(home)
	snap.QuickTunnel = parseCloudflaredQuickTunnel(snap.Process.Cmdline)
	snap.Tunnels = captureCloudflaredTunnels(home)
	snap.Config = readCloudflaredConfigFile(home)
	snap.Setup = buildCloudflaredSetup(home)

	return snap, true, nil
}

func cloudflaredRunningGate(home string) bool {
	if home == "" {
		return false
	}
	pid, cmdline, ok := cloudflaredProcInfo(home)
	if !ok || pid <= 0 {
		return false
	}
	if !cloudflaredHomeSpecificEvidence(home, cmdline) {
		return false
	}
	env := cloudflaredCommandEnv(home)
	if _, err := resolveCloudflaredBinary(home, env); err != nil {
		return false
	}
	return true
}

// cloudflaredHomeSpecificEvidence is true when cloudflared plausibly belongs to home,
// not merely because the host has cloudflared installed or running.
func cloudflaredHomeSpecificEvidence(home, cmdline string) bool {
	if _, ok := cloudflaredBinaryInHomeBin(home); ok {
		return true
	}
	if cloudflaredCmdlineFromStub(home) != "" {
		return true
	}
	if cmdline != "" && strings.Contains(cmdline, home) {
		return true
	}
	return cloudflaredHasHomeConfigFootprint(home)
}

func cloudflaredHasHomeConfigFootprint(home string) bool {
	if cloudflaredHasTunnelCredentials(home) {
		return true
	}
	configPath := filepath.Join(home, ".cloudflared", "config.yml")
	if _, err := os.Stat(configPath); err == nil {
		return true
	}
	return false
}

func cloudflaredVersionOutput(home string) (string, json.RawMessage, error) {
	textOut, err := runCloudflaredCommand(home, "version")
	if err != nil {
		return "", nil, err
	}
	jsonOut, err := runCloudflaredCommand(home, "version", "--json")
	if err != nil {
		return string(textOut), nil, nil
	}
	trimmed := bytes.TrimSpace(jsonOut)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return string(textOut), nil, nil
	}
	return string(textOut), json.RawMessage(trimmed), nil
}

func runCloudflaredCommand(home string, args ...string) ([]byte, error) {
	env := cloudflaredCommandEnv(home)
	bin, err := resolveCloudflaredBinary(home, env)
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

	cmd := cloudflaredExecCommand(execBin, args)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func cloudflaredExecCommand(bin string, args []string) *exec.Cmd {
	if isShellScript(bin) {
		return exec.Command("sh", append([]string{bin}, args...)...)
	}
	return exec.Command(bin, args...)
}

func resolveCloudflaredBinary(home string, env []string) (string, error) {
	for _, candidateHome := range tailscaleHomeCandidates(home) {
		if bin, ok := cloudflaredBinaryInHomeBin(candidateHome); ok {
			return bin, nil
		}
	}
	return lookPathInEnv("cloudflared", env)
}

func cloudflaredBinaryInHomeBin(home string) (string, bool) {
	candidate := filepath.Join(home, "bin", "cloudflared")
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return "", false
	}
	return candidate, true
}

func cloudflaredCommandEnv(home string) []string {
	env := os.Environ()
	if home == "" {
		return env
	}
	binDir := filepath.Join(home, "bin")
	if st, err := os.Stat(filepath.Join(binDir, "cloudflared")); err != nil || st.IsDir() {
		return env
	}
	return prependPathEnv(env, binDir)
}

func discoverCloudflaredProcess(home string) CloudflaredProcessInfo {
	info := CloudflaredProcessInfo{}
	if pid, cmdline, ok := cloudflaredProcInfo(home); ok {
		info.PID = pid
		if cmdline != "" {
			info.Cmdline = cmdline
		}
	}
	if info.Cmdline == "" {
		if cmdline := cloudflaredCmdlineFromStub(home); cmdline != "" {
			info.Cmdline = cmdline
		}
	}
	return info
}

func cloudflaredProcInfo(home string) (int, string, bool) {
	env := cloudflaredCommandEnv(home)
	cmd := exec.Command("pgrep", "cloudflared")
	cmd.Env = env
	pidOut, err := cmd.Output()
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
		cmdline := cloudflaredCmdlineFromStub(home)
		return pid, cmdline, true
	}
	return pid, strings.ReplaceAll(string(raw), "\x00", " "), true
}

func cloudflaredCmdlineFromStub(home string) string {
	path := filepath.Join(home, ".doctest-cloudflared.cmdline")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func parseCloudflaredQuickTunnel(cmdline string) CloudflaredQuickTunnelInfo {
	info := CloudflaredQuickTunnelInfo{}
	if cmdline == "" {
		return info
	}
	args := strings.Fields(cmdline)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--url="):
			info.URL = strings.TrimPrefix(arg, "--url=")
		case arg == "--url" && i+1 < len(args):
			i++
			info.URL = args[i]
		case strings.HasPrefix(arg, "--hostname="):
			info.Hostname = strings.TrimPrefix(arg, "--hostname=")
		case arg == "--hostname" && i+1 < len(args):
			i++
			info.Hostname = args[i]
		}
	}
	return info
}

func cloudflaredHasTunnelCredentials(home string) bool {
	certPath := filepath.Join(home, ".cloudflared", "cert.pem")
	if _, err := os.Stat(certPath); err == nil {
		return true
	}
	return false
}

func captureCloudflaredTunnels(home string) CloudflaredTunnelsInfo {
	if !cloudflaredHasTunnelCredentials(home) {
		return CloudflaredTunnelsInfo{
			Available: false,
			Error:     cloudflaredTunnelListErr,
			Items:     []json.RawMessage{},
		}
	}

	out, err := runCloudflaredCommand(home, "tunnel", "list", "--output", "json")
	if err != nil {
		return CloudflaredTunnelsInfo{
			Available: false,
			Error:     formatCloudflaredCommandError(err, out),
			Items:     []json.RawMessage{},
		}
	}

	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 || string(trimmed) == "[]" {
		return CloudflaredTunnelsInfo{
			Available: true,
			Items:     []json.RawMessage{},
		}
	}

	var items []json.RawMessage
	if err := json.Unmarshal(trimmed, &items); err != nil {
		return CloudflaredTunnelsInfo{
			Available: false,
			Error:     fmt.Sprintf("parse tunnel list json: %v", err),
			Items:     []json.RawMessage{},
		}
	}
	return CloudflaredTunnelsInfo{
		Available: true,
		Items:     items,
	}
}

func formatCloudflaredCommandError(err error, out []byte) string {
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

func readCloudflaredConfigFile(home string) CloudflaredConfigFileInfo {
	path := filepath.Join(home, ".cloudflared", "config.yml")
	info := CloudflaredConfigFileInfo{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		return info
	}
	info.Present = true
	info.RedactedYAML = redactCloudflaredConfigYAML(string(data))
	return info
}

func redactCloudflaredConfigYAML(raw string) string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "tunnel:"):
			lines = append(lines, redactCloudflaredYAMLKey(line, "tunnel"))
		case strings.HasPrefix(trimmed, "credentials-file:"):
			lines = append(lines, redactCloudflaredYAMLKey(line, "credentials-file"))
		default:
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func redactCloudflaredYAMLKey(line, key string) string {
	if idx := strings.Index(line, key+":"); idx >= 0 {
		prefix := line[:idx+len(key)+1]
		return prefix + " <redacted>"
	}
	return key + ": <redacted>"
}

func readCloudflaredHistoryLines(home, name string) []string {
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
		if cloudflaredHistoryLineRE.MatchString(line) {
			lines = append(lines, line)
		}
	}
	return lines
}

func buildCloudflaredSetup(home string) CloudflaredSetupInfo {
	return CloudflaredSetupInfo{
		BashHistory: readCloudflaredHistoryLines(home, ".bash_history"),
		ZshHistory:  readCloudflaredHistoryLines(home, ".zsh_history"),
	}
}

func marshalCloudflaredConfigSnapshot(snap *CloudflaredConfigSnapshot) ([]byte, error) {
	if snap == nil {
		return nil, nil
	}
	return json.MarshalIndent(snap, "", "  ")
}

func formatCloudflaredSummaryLinesForHome(home string) []string {
	for _, candidate := range tailscaleHomeCandidates(home) {
		if snap, included, err := buildCloudflaredConfigSnapshotFn(candidate); err == nil && included && snap != nil {
			return formatCloudflaredSummaryLines(snap)
		}
	}
	return nil
}

func captureCloudflaredConfigForHome(home string) (*CloudflaredConfigSnapshot, bool, error) {
	for _, candidate := range tailscaleHomeCandidates(home) {
		if snap, included, err := buildCloudflaredConfigSnapshotFn(candidate); err != nil {
			return nil, false, err
		} else if included && snap != nil {
			return snap, true, nil
		}
	}
	return nil, false, nil
}

func formatCloudflaredSummaryLines(snap *CloudflaredConfigSnapshot) []string {
	if snap == nil || !snap.Running {
		return nil
	}

	mode := cloudflaredModeLabel(snap)
	target := cloudflaredTargetLabel(snap)
	if target == "" {
		target = "-"
	}

	lines := []string{
		cloudflaredDryRunHeader,
		fmt.Sprintf("    captured_at: %s  (running)", formatMetaCapturedAt(snap.CapturedAt)),
		cloudflaredTableColumnHdr,
		fmt.Sprintf("    %-21s %-14s %s", snap.VersionInfo.Text, mode, target),
		"",
		"    DAEMON",
	}
	if snap.Process.Cmdline != "" {
		lines = append(lines, "      "+snap.Process.Cmdline)
	} else {
		lines = append(lines, "      (unknown)")
	}

	lines = append(lines, "", "    CONFIG")
	if snap.Config.Present {
		lines = append(lines, fmt.Sprintf("      %s  (present, redacted)", snap.Config.Path))
	} else if snap.Config.Path != "" {
		lines = append(lines, fmt.Sprintf("      %s  (absent)", snap.Config.Path))
	} else {
		lines = append(lines, "      (unknown)")
	}

	lines = append(lines, "", "    SHELL HISTORY (cloudflared)")
	for _, line := range snap.Setup.BashHistory {
		lines = append(lines, "      [bash] "+line)
	}
	for _, line := range snap.Setup.ZshHistory {
		lines = append(lines, "      [zsh]  "+line)
	}
	return lines
}

func cloudflaredModeLabel(snap *CloudflaredConfigSnapshot) string {
	if snap == nil {
		return "tunnel"
	}
	if snap.QuickTunnel.URL != "" {
		return "quick-tunnel"
	}
	if snap.QuickTunnel.Hostname != "" {
		return "hostname-tunnel"
	}
	return "tunnel"
}

func cloudflaredTargetLabel(snap *CloudflaredConfigSnapshot) string {
	if snap == nil {
		return ""
	}
	if snap.QuickTunnel.URL != "" {
		return snap.QuickTunnel.URL
	}
	if snap.QuickTunnel.Hostname != "" {
		return snap.QuickTunnel.Hostname
	}
	return ""
}