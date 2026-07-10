# macOS Remote Menu Bar Backup Progress Window Doctests

Pure-function tests for **Backup Now progress window** helpers
(`macosapp/menubar`) and one-shot-when-disabled policy, plus Swift source
contracts for the remote menu-bar app (`ai-critic-remote-macos`).

Sibling of `tests/macos-menubar-backup/` (paths/schedule/status/recent). This tree
covers **visible progress UX** and the policy that **Backup Now is a one-shot
independent of Enable**. No live network; no UI automation in default leaves.

# DSN (Domain Specific Notion)

**Participants**

- **Backup progress helpers (`macosapp/menubar`)** — pure Go functions (mirrored
  by Swift `BackupMenuFormatter`) for:
  - `CanRunBackupNow` — menu enablement for Backup Now (endpoint, not running,
    non-empty server name; **does not take** task `enabled`);
  - `ShouldShowBackupProgressWindow` — whether this invocation opens the
    progress window (false only when triggered by hourly schedule);
  - `FormatBackupProgress*` — sealed human lines for SSE frames, local phases,
    guards, success/failure footers, and window title.
- **Backup run session (UI, conceptual)** — append-only lines, outcome
  running|success|failed, optional path/size/error; `showWindow` true for
  manual Backup Now and enable-immediate, false for schedule ticks.
- **Progress window (`BackupProgressWindow` family)** — AppKit monospaced
  scrollable log; title `Backup: {server}` / `Backup: (no server)`;
  `makeKeyAndOrderFront`; close does not cancel the job (v1).
- **Remote macOS menu bar (`ai-critic-remote-macos`)** — Backup submenu; Backup
  Now…; enable path that may run immediately; hourly tick via
  `triggeredBySchedule`.
- **Machine backup client** — stream SSE (`section` / `progress` / `log` /
  `error` / `done` with `archive_token`) then download archive; must surface
  intermediate frames to the progress window (not only keep the token).
- **Test harness** — invokes Go helpers with fixed inputs, or greps remote
  Swift sources; no UI automation, no network download.

**Behaviors (sealed)**

- `CanRunBackupNow(hasEndpoint, running, serverName)`:
  - true only when `hasEndpoint && !running && serverName != ""` (after trim);
  - **independent of** periodic task `enabled` (one-shot always allowed when ready).
- `ShouldShowBackupProgressWindow(triggeredBySchedule)`:
  - `false` when `triggeredBySchedule` (hourly silent);
  - `true` otherwise (manual Backup Now, enable-immediate).
- Progress lines (exact sealed strings):
  - start header: `Machine backup — {server}`
  - started at: `Started {YYYY-MM-DD HH:MM:SS}` (wall clock of the given time)
  - window title: `Backup: {server}` or `Backup: (no server)` when empty
  - SSE section: `[section] {message}`
  - SSE progress: `[progress] {name} {status}`; with detail:
    `[progress] {name} {status} — {detail}`
  - SSE log (verbatim style): `{message}` only (no `[log]` prefix)
  - SSE error: `ERROR: {message}`
  - SSE done (empty message): `[done] archive ready`; non-empty: `[done] {message}`
  - download start: `Downloading archive…`
  - wrote: `Wrote {path} ({human size})` (same size units as recent list: `42 MB`)
  - success footer: `Status: Success`
  - failure footer: `Status: Failed`
  - guards: `ERROR: not configured` / `ERROR: no server selected`
- Swift contracts: progress window type/open; manual + enable-immediate open
  window; Backup Now `.disabled` only on endpoint/running (not `backupEnabled`);
  stream progress consumed via callback/onEvent; schedule ticks do not open window.

## Version

0.0.2

## Decision Tree

```
[macos-menubar-backup-progress]
 |
 +-- can-run/                              (GROUP)  CanRunBackupNow (ignores enabled)
 |    +-- when-disabled/                   (LEAF)   ready + task off → true (one-shot)
 |    +-- when-enabled/                    (LEAF)   ready + task on  → true
 |    +-- no-endpoint/                     (LEAF)   false
 |    +-- already-running/                 (LEAF)   false
 |    +-- empty-server/                    (LEAF)   false
 |
 +-- show-window/                          (GROUP)  ShouldShowBackupProgressWindow
 |    +-- not-schedule/                    (LEAF)   manual / enable-immediate → true
 |    +-- schedule/                        (LEAF)   hourly tick → false
 |
 +-- format/                               (GROUP)  FormatBackupProgress* sealed lines
 |    +-- start-header/                    (LEAF)   Machine backup — {server}
 |    +-- started-at/                      (LEAF)   Started 2026-07-10 15:00:00
 |    +-- window-title/                    (GROUP)  Backup: …
 |    |    +-- with-server/                (LEAF)   Backup: foo.example.com
 |    |    +-- no-server/                  (LEAF)   Backup: (no server)
 |    +-- section/                         (LEAF)   [section] {message}
 |    +-- progress-name-status/            (LEAF)   [progress] name status
 |    +-- progress-with-detail/            (LEAF)   … — detail
 |    +-- log-verbatim/                    (LEAF)   message only
 |    +-- error/                           (LEAF)   ERROR: {message}
 |    +-- done/                            (LEAF)   [done] archive ready
 |    +-- download-start/                  (LEAF)   Downloading archive…
 |    +-- wrote/                           (LEAF)   Wrote path (42 MB)
 |    +-- status-success/                  (LEAF)   Status: Success
 |    +-- status-failed/                   (LEAF)   Status: Failed
 |    +-- guard-not-configured/            (LEAF)   ERROR: not configured
 |    +-- guard-no-server/                 (LEAF)   ERROR: no server selected
 |
 +-- client/                               (GROUP)  remote Swift source contracts
      +-- progress-window/                 (LEAF)   BackupProgressWindow / open + append
      +-- manual-shows-window/             (LEAF)   Backup Now opens window
      +-- enable-immediate-shows-window/   (LEAF)   enable-triggered run opens window
      +-- not-gated-on-enabled/            (LEAF)   .disabled endpoint|running only
      +-- stream-progress-consumed/        (LEAF)   SSE progress not token-only
      +-- schedule-silent/                 (LEAF)   schedule path no window
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `can-run/when-disabled` | One-shot when task disabled → `CanRunBackupNow=true` |
| 2 | `can-run/when-enabled` | Still true when task enabled |
| 3 | `can-run/no-endpoint` | No endpoint → false |
| 4 | `can-run/already-running` | Already running → false |
| 5 | `can-run/empty-server` | Empty server name → false |
| 6 | `show-window/not-schedule` | Manual / enable-immediate → show window |
| 7 | `show-window/schedule` | Hourly schedule → no window |
| 8 | `format/start-header` | `Machine backup — foo.example.com` |
| 9 | `format/started-at` | `Started 2026-07-10 15:00:00` |
| 10 | `format/window-title/with-server` | `Backup: foo.example.com` |
| 11 | `format/window-title/no-server` | `Backup: (no server)` |
| 12 | `format/section` | `[section] Collecting files` |
| 13 | `format/progress-name-status` | `[progress] home ok` |
| 14 | `format/progress-with-detail` | `[progress] home ok — 12 files` |
| 15 | `format/log-verbatim` | verbatim log body only |
| 16 | `format/error` | `ERROR: stream failed` |
| 17 | `format/done` | `[done] archive ready` |
| 18 | `format/download-start` | `Downloading archive…` |
| 19 | `format/wrote` | `Wrote … (42 MB)` |
| 20 | `format/status-success` | `Status: Success` |
| 21 | `format/status-failed` | `Status: Failed` |
| 22 | `format/guard-not-configured` | `ERROR: not configured` |
| 23 | `format/guard-no-server` | `ERROR: no server selected` |
| 24 | `client/progress-window` | Progress window type + open/append present |
| 25 | `client/manual-shows-window` | Manual Backup Now opens progress window |
| 26 | `client/enable-immediate-shows-window` | Enable-triggered immediate run shows window |
| 27 | `client/not-gated-on-enabled` | Backup Now not disabled by `backupEnabled` |
| 28 | `client/stream-progress-consumed` | Stream frames consumed for display |
| 29 | `client/schedule-silent` | `triggeredBySchedule: true` does not open window |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| when-disabled | can_run | endpoint+name, !running (task off story) | true |
| when-enabled | can_run | same ready inputs (task on story) | true |
| no-endpoint | can_run | hasEndpoint=false | false |
| already-running | can_run | running=true | false |
| empty-server | can_run | serverName="" | false |
| not-schedule | show_window | triggeredBySchedule=false | true |
| schedule | show_window | triggeredBySchedule=true | false |
| start-header | format_start_header | server=foo.example.com | sealed header |
| started-at | format_started_at | fixed UTC wall | sealed Started line |
| with-server | format_window_title | name set | `Backup: foo…` |
| no-server | format_window_title | name empty | `Backup: (no server)` |
| section | format_section | message | `[section] …` |
| progress-name-status | format_progress | name+status | `[progress] …` |
| progress-with-detail | format_progress | +detail | with ` — ` |
| log-verbatim | format_log | message | message only |
| error | format_error | message | `ERROR: …` |
| done | format_done | empty message | `[done] archive ready` |
| download-start | format_download_start | — | `Downloading archive…` |
| wrote | format_wrote | path + 42MiB | Wrote + size |
| status-success | format_status_success | — | `Status: Success` |
| status-failed | format_status_failed | — | `Status: Failed` |
| guard-not-configured | format_guard | not_configured | `ERROR: not configured` |
| guard-no-server | format_guard | no_server | `ERROR: no server selected` |
| progress-window | client | remote Swift | window type/open |
| manual-shows-window | client | remote Swift | show on Backup Now |
| enable-immediate-shows-window | client | remote Swift | show on enable-immediate |
| not-gated-on-enabled | client | remote Swift | no enabled gate |
| stream-progress-consumed | client | remote Swift | progress callback |
| schedule-silent | client | remote Swift | no window on schedule |

## How to Run

```sh
doctest vet ./tests/macos-menubar-backup-progress
doctest test ./tests/macos-menubar-backup-progress/...
```

Existing sealed tree remains unchanged:

```sh
doctest test ./tests/macos-menubar-backup/...
```

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

// Fixed clock for pure time-format leaves.
const defaultNowRFC3339 = "2026-07-10T15:00:00Z"

type Request struct {
	Op string

	// can_run
	HasEndpoint bool
	Running     bool
	ServerName  string
	// Enabled is documentation-only for can-run leaves (policy story); pure
	// CanRunBackupNow does not take it. Leaves still set it to record intent.
	Enabled bool

	// show_window
	TriggeredBySchedule bool

	// format inputs
	Message     string
	ProgressName   string
	ProgressStatus string
	ProgressDetail string
	Path        string
	SizeBytes   int64
	GuardReason string // not_configured | no_server
	TimeRFC3339 string // for started-at

	// client
	ClientLeaf string
}

type Response struct {
	CanRun     bool
	ShowWindow bool

	Line  string // formatted progress / title / footer line
	Title string // window title (also may use Line)

	// client contract flags
	HasProgressWindow            bool
	ManualShowsWindow            bool
	EnableImmediateShowsWindow   bool
	BackupNowNotGatedOnEnabled   bool
	StreamProgressConsumed       bool
	ScheduleSilent               bool
	SwiftSourcesChecked          []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "can_run":
		// One-shot policy: enabled is intentionally NOT passed to the helper.
		resp.CanRun = menubar.CanRunBackupNow(req.HasEndpoint, req.Running, req.ServerName)
	case "show_window":
		resp.ShowWindow = menubar.ShouldShowBackupProgressWindow(req.TriggeredBySchedule)
	case "format_start_header":
		resp.Line = menubar.FormatBackupProgressStartHeader(req.ServerName)
	case "format_started_at":
		ts, err := parseRFC3339(req.TimeRFC3339)
		if err != nil {
			return nil, err
		}
		resp.Line = menubar.FormatBackupProgressStartedAt(ts)
	case "format_window_title":
		resp.Title = menubar.FormatBackupProgressWindowTitle(req.ServerName)
		resp.Line = resp.Title
	case "format_section":
		resp.Line = menubar.FormatBackupProgressSection(req.Message)
	case "format_progress":
		resp.Line = menubar.FormatBackupProgressFrame(req.ProgressName, req.ProgressStatus, req.ProgressDetail)
	case "format_log":
		resp.Line = menubar.FormatBackupProgressLog(req.Message)
	case "format_error":
		resp.Line = menubar.FormatBackupProgressError(req.Message)
	case "format_done":
		resp.Line = menubar.FormatBackupProgressDone(req.Message)
	case "format_download_start":
		resp.Line = menubar.FormatBackupProgressDownloadStart()
	case "format_wrote":
		resp.Line = menubar.FormatBackupProgressWrote(req.Path, req.SizeBytes)
	case "format_status_success":
		resp.Line = menubar.FormatBackupProgressStatusSuccess()
	case "format_status_failed":
		resp.Line = menubar.FormatBackupProgressStatusFailed()
	case "format_guard":
		resp.Line = menubar.FormatBackupProgressGuardError(req.GuardReason)
	case "client":
		return runClientContract(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
	return resp, nil
}

func parseRFC3339(s string) (time.Time, error) {
	if s == "" {
		s = defaultNowRFC3339
	}
	return time.Parse(time.RFC3339, s)
}

func runClientContract(t *testing.T, req *Request, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}
	remoteApp := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-remote-macos", "AICriticApp.swift")
	sharedDir := filepath.Join(moduleRoot, "macos-ai-critic", "Shared")
	remoteDir := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-remote-macos")

	remoteSrc, err := os.ReadFile(remoteApp)
	if err != nil {
		return nil, fmt.Errorf("read remote AICriticApp.swift: %w", err)
	}
	remoteStr := string(remoteSrc)
	resp.SwiftSourcesChecked = []string{remoteApp}

	combined := remoteStr
	for _, dir := range []string{remoteDir, sharedDir} {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || info == nil || info.IsDir() {
				return walkErr
			}
			if !strings.HasSuffix(path, ".swift") {
				return nil
			}
			if path == remoteApp {
				return nil
			}
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			combined += "\n" + string(b)
			resp.SwiftSourcesChecked = append(resp.SwiftSourcesChecked, path)
			return nil
		})
	}

	resp.HasProgressWindow = hasBackupProgressWindow(combined)
	resp.ManualShowsWindow = hasManualBackupShowsWindow(combined)
	resp.EnableImmediateShowsWindow = hasEnableImmediateShowsWindow(combined)
	resp.BackupNowNotGatedOnEnabled = hasBackupNowNotGatedOnEnabled(combined)
	resp.StreamProgressConsumed = hasStreamProgressConsumed(combined)
	resp.ScheduleSilent = hasScheduleSilent(combined)

	switch req.ClientLeaf {
	case "progress-window",
		"manual-shows-window",
		"enable-immediate-shows-window",
		"not-gated-on-enabled",
		"stream-progress-consumed",
		"schedule-silent":
		// fields populated above
	default:
		return nil, fmt.Errorf("unknown client leaf %q", req.ClientLeaf)
	}
	return resp, nil
}

func hasBackupProgressWindow(src string) bool {
	// Type / enum / open helper for append-only progress UI (LogStreamWindow family).
	if strings.Contains(src, "BackupProgressWindow") {
		return true
	}
	if regexp.MustCompile(`(?i)(struct|class|enum)\s+BackupProgress`).MatchString(src) {
		return true
	}
	// open + monospaced progress for backup
	openProgress := regexp.MustCompile(`(?i)(openBackupProgress|showBackupProgress|BackupProgress.*open)`).MatchString(src)
	return openProgress
}

func hasManualBackupShowsWindow(src string) bool {
	// Backup Now path opens progress window (not schedule-only).
	hasBackupNow := strings.Contains(src, "Backup Now")
	if !hasBackupNow {
		return false
	}
	// runBackupNow default / false schedule shows window, or explicit showWindow: true
	opens := regexp.MustCompile(`(?is)(runBackupNow\s*\(|Backup Now)[\s\S]{0,800}(BackupProgressWindow|showWindow\s*[:=]\s*true|ShouldShowBackupProgressWindow|openBackupProgress|showBackupProgress)`).MatchString(src)
	// Or helper: if !triggeredBySchedule { open window }
	opensAlt := regexp.MustCompile(`(?is)(!triggeredBySchedule|triggeredBySchedule\s*==\s*false|showWindow)[\s\S]{0,400}(BackupProgressWindow|makeKeyAndOrderFront|open\()`).MatchString(src)
	return opens || opensAlt || (hasBackupProgressWindow(src) && regexp.MustCompile(`(?is)func\s+runBackupNow[\s\S]{0,1000}BackupProgress`).MatchString(src))
}

func hasEnableImmediateShowsWindow(src string) bool {
	// Enable path that kicks immediate run must open the same progress window
	// (showWindow true / triggeredBySchedule false for that call).
	enableThenRun := regexp.MustCompile(`(?is)(setBackupEnabled|enable.*backup|shouldRunOnEnable|shouldRunNow)[\s\S]{0,1000}runBackupNow`).MatchString(src)
	if !enableThenRun {
		// still require that enable-triggered calls do not force schedule-silent only
		return false
	}
	// Prefer explicit non-schedule show: runBackupNow() default, or triggeredBySchedule: false, or showWindow: true
	goodCall := regexp.MustCompile(`(?is)(shouldRunNow|ShouldRunOnEnable|setBackupEnabled\s*\(\s*true)[\s\S]{0,800}runBackupNow\s*\(\s*(triggeredBySchedule\s*:\s*false|showWindow\s*:\s*true)?\s*\)`).MatchString(src)
	// Reject only-schedule-true on enable path without a false/show path nearby
	onlyScheduleTrue := regexp.MustCompile(`(?is)(shouldRunNow|setBackupEnabled)[\s\S]{0,600}runBackupNow\s*\(\s*triggeredBySchedule\s*:\s*true\s*\)`).MatchString(src)
	if goodCall && !onlyScheduleTrue {
		return true
	}
	// Accept showWindow policy helper used for enable-immediate
	if regexp.MustCompile(`(?is)(shouldRunNow|onEnable)[\s\S]{0,600}(ShouldShowBackupProgressWindow\s*\(\s*false|showWindow\s*[:=]\s*true|triggeredBySchedule\s*:\s*false)`).MatchString(src) {
		return true
	}
	// Strong signal: enable path + progress window open in same function region
	return regexp.MustCompile(`(?is)func\s+setBackupEnabled[\s\S]{0,1000}(BackupProgressWindow|showWindow\s*[:=]\s*true|triggeredBySchedule\s*:\s*false)`).MatchString(src)
}

func hasBackupNowNotGatedOnEnabled(src string) bool {
	// Button("Backup Now…") .disabled must not reference backupEnabled as a gate.
	// Accept disabled only on !hasEndpoint / backupRunning (and similar).
	btn := regexp.MustCompile(`(?is)Button\s*\(\s*"Backup Now[^\"]*"\s*\)[\s\S]{0,400}\.disabled\s*\(([^)]+)\)`)
	m := btn.FindStringSubmatch(src)
	if len(m) < 2 {
		// fallback: any disabled near Backup Now without backupEnabled
		region := regexp.MustCompile(`(?is)Backup Now[\s\S]{0,500}\.disabled\s*\(([^)]+)\)`)
		m = region.FindStringSubmatch(src)
		if len(m) < 2 {
			return false
		}
	}
	expr := m[1]
	if regexp.MustCompile(`(?i)backupEnabled|backup\.enabled|st\.enabled`).MatchString(expr) {
		return false
	}
	// Must gate on endpoint and/or running
	hasEndpointGate := regexp.MustCompile(`(?i)hasEndpoint|isConfigured|endpoint`).MatchString(expr)
	hasRunningGate := regexp.MustCompile(`(?i)backupRunning|running`).MatchString(expr)
	return hasEndpointGate || hasRunningGate
}

func hasStreamProgressConsumed(src string) bool {
	// Intermediate SSE frames used for UI, not only archive_token extraction.
	tokenOnly := strings.Contains(src, "archive_token") || strings.Contains(src, "archiveToken")
	_ = tokenOnly
	// Progress callback / onEvent / FormatBackupProgress / append line from frames
	if regexp.MustCompile(`(?i)(onProgress|onEvent|progressHandler|progressCallback|BackupProgressEvent|FormatBackupProgress)`).MatchString(src) {
		return true
	}
	if regexp.MustCompile(`(?is)(type\s*==\s*"progress"|case\s+"progress"|\"section\").*?(append|onProgress|formatBackupProgress)`).MatchString(src) {
		return true
	}
	// downloadBackupArchive with trailing progress closure / AsyncStream of events
	if regexp.MustCompile(`(?is)downloadBackupArchive[\s\S]{0,200}(onProgress|progress:|events:|AsyncStream|callback)`).MatchString(src) {
		return true
	}
	if regexp.MustCompile(`(?is)streamBackup[\s\S]{0,400}(onProgress|yield|progress)`).MatchString(src) {
		return true
	}
	return false
}

func hasScheduleSilent(src string) bool {
	// Schedule path uses triggeredBySchedule: true and must not open window for that case.
	hasScheduleCall := regexp.MustCompile(`(?is)(checkBackupDue|triggeredBySchedule\s*:\s*true)[\s\S]{0,200}runBackupNow|runBackupNow\s*\(\s*triggeredBySchedule\s*:\s*true\s*\)`).MatchString(src)
	if !hasScheduleCall {
		// require explicit schedule-true somewhere for due/tick
		hasScheduleCall = strings.Contains(src, "triggeredBySchedule: true") || strings.Contains(src, "triggeredBySchedule:true")
	}
	// Policy helper or branch: if triggeredBySchedule { skip window }
	silentPolicy := regexp.MustCompile(`(?is)(ShouldShowBackupProgressWindow|triggeredBySchedule)[\s\S]{0,300}(false|!showWindow|skip|return)`).MatchString(src)
	// Or: only open when !triggeredBySchedule
	openOnlyManual := regexp.MustCompile(`(?is)if\s+(!triggeredBySchedule|triggeredBySchedule\s*==\s*false|showWindow)[\s\S]{0,200}(BackupProgressWindow|openBackupProgress|showBackupProgress|makeKeyAndOrderFront)`).MatchString(src)
	return hasScheduleCall && (silentPolicy || openOnlyManual || hasBackupProgressWindow(src))
}

func findModuleRoot() (string, error) {
	if root := os.Getenv("DOCTEST_ROOT"); root != "" {
		for dir := root; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
```
