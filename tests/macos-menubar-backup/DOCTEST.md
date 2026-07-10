# macOS Remote Menu Bar Periodic Machine Backup Doctests

Pure-function tests for periodic machine-backup helpers (`macosapp/menubar`) used by
the **remote** menu-bar app (`ai-critic-remote-macos`), plus Swift source contracts
for the Backup submenu (status, enable/disable, recent, reveal, backup now).

Default task is **OFF**. Interval is **1 hour**. Archives are **`.tar.xz`** under
`~/.backup/ai-critic/<server-name>/`. No live network in default leaves.

# DSN (Domain Specific Notion)

**Participants**

- **Backup helpers (`macosapp/menubar`)** — pure Go functions (mirrored by Swift)
  for server-scoped paths, status/entry formatting, enable-on-run and due
  policy, recent-list ordering, retention prune, and Enable/Disable item
  enablement.
- **Per-server task state** — local JSON (`periodic-backup.json` or equivalent)
  keyed by server-name; UI always reflects the **current** default domain only.
- **Local archive store** — `~/.backup/ai-critic/<server-name>/machine-backup-<utc-ts>.tar.xz`
  written by the app after stream+token download (same API as CLI machine backup).
- **Remote macOS menu bar (`ai-critic-remote-macos`)** — Backup submenu under the
  menubar dropdown, scoped to the active Server selection (same context as
  Services/Terminals).
- **Remote machine backup API** — `POST …/machine/backup/stream` → `archive_token`
  in `done` frame → GET archive by token (no live calls in this tree).
- **Test harness** — invokes Go helpers with fixed `now` and entry structs, or
  inspects remote Swift sources; no UI automation, no network download.

**Behaviors**

- **Default**: task `enabled=false`; status title `Status: Off`.
- **Interval**: 3600 seconds (1 hour).
- `ServerNameFromURL(url)`: host of the resolved server URL, filesystem-safe
  (no scheme, path, or trailing slash). Example: `https://foo.example.com/` →
  `foo.example.com`.
- `BackupDir(home, serverName)`: `{home}/.backup/ai-critic/{serverName}`.
- `BackupArchiveFilename(utc)`: `machine-backup-<YYYYMMDD-HHMMSS>.tar.xz` (UTC).
- `ShouldRunOnEnable(lastFinished, now, interval)`:
  - never ran (zero lastFinished) → **true** (run immediately);
  - last finish ≤ interval ago → **false** (enable only);
  - last finish > interval ago → **true**.
- `ShouldRunDue(enabled, running, nextRunAt, now)`:
  - disabled → false; already running → false (no overlap);
  - enabled, not running, `nextRunAt <= now` → true.
- `FormatBackupStatusTitle(status, now)` sealed strings:
  - off / disabled → `Status: Off`
  - on + running → `Status: On · Running`
  - on + error → `Status: On · Error · {rel} ago` (relative from last finish)
  - on + idle with last/next → `Status: On · last {rel} · next in {rel}`
  - relative past: `Nm ago` / `Nh ago` / `Nd ago`; future: `in Nm` / `in Nh` / `in Nd`
  - separator is middle-dot with spaces: ` · `
- `FormatBackupEntry(entry, now)`: `{rel past} · {human size}` e.g. `12m ago · 42 MB`
  (binary units, whole numbers for exact MiB multiples; space before unit).
- `FormatBackupRecentEmptyLabel()`: `No recent backups`.
- Recent list: entries ordered **newest first** by modTime.
- `PruneBackupFiles(entries, now)` (defaults keepTodayN=10, dailyDays=7):
  - **today** (local calendar of `now`): keep newest **10**, delete older today;
  - **past days within 7 calendar days**: keep **1** per day (newest of that day);
  - **older / outside window**: delete all.
- Status nested menu children: only **Enable** and **Disable**.
  - task off → Enable active, Disable inactive;
  - task on → Disable active, Enable inactive.
- Swift (remote app): Backup menu present; Status nested Enable/Disable; default
  off on launch (no auto-enable); download path uses stream + `archive_token`
  like CLI (not a one-shot non-token path only).

## Version

0.0.2

## Decision Tree

```
[macos-menubar-backup]
 |
 +-- path/                                 (GROUP)  server scope + archive naming
 |    +-- server-name-from-url/            (LEAF)   host only, no scheme/slash
 |    +-- backup-dir/                      (LEAF)   {home}/.backup/ai-critic/{name}
 |    +-- archive-extension/               (LEAF)   machine-backup-*.tar.xz (UTC)
 |
 +-- schedule/                             (GROUP)  enable + due + interval
 |    +-- enable/                          (GROUP)  ShouldRunOnEnable
 |    |    +-- never-ran/                  (LEAF)   true → immediate run
 |    |    +-- recent-finish/              (LEAF)   30m ago → false
 |    |    +-- stale-finish/               (LEAF)   2h ago → true
 |    +-- due/                             (GROUP)  ShouldRunDue
 |    |    +-- enabled-due/                (LEAF)   enabled, next<=now, not running
 |    |    +-- running/                    (LEAF)   no overlap
 |    |    +-- disabled/                   (LEAF)   disabled → false
 |    +-- interval/                        (GROUP)  1h constant
 |         +-- one-hour/                   (LEAF)   3600s
 |
 +-- status/                               (GROUP)  FormatBackupStatusTitle
 |    +-- off/                             (LEAF)   Status: Off (default off)
 |    +-- on-idle/                         (LEAF)   On · last … · next in …
 |    +-- on-running/                      (LEAF)   Status: On · Running
 |    +-- on-error/                        (LEAF)   Status: On · Error · rel
 |
 +-- recent/                               (GROUP)  list + format
 |    +-- empty/                           (LEAF)   No recent backups
 |    +-- sorted-newest-first/             (LEAF)   mtime desc
 |    +-- format-entry/                    (LEAF)   12m ago · 42 MB
 |
 +-- retention/                            (GROUP)  PruneBackupFiles
 |    +-- today-over-limit/                (LEAF)   keep newest 10 today
 |    +-- past-day-multiple/               (LEAF)   keep 1 per past day
 |    +-- older-than-7-days/               (LEAF)   delete
 |    +-- mixed/                           (LEAF)   all three rules
 |
 +-- menu/                                 (GROUP)  Status children + enable gating
 |    +-- status-children/                 (LEAF)   only Enable, Disable
 |    +-- when-off/                        (LEAF)   Enable active
 |    +-- when-on/                         (LEAF)   Disable active
 |
 +-- client/                               (GROUP)  remote Swift source contracts
      +-- backup-menu/                     (LEAF)   Backup submenu present
      +-- status-enable-disable/           (LEAF)   nested Status Enable/Disable
      +-- default-off/                     (LEAF)   no auto-enable on launch
      +-- stream-token-download/           (LEAF)   stream + archive_token path
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `path/server-name-from-url` | `https://foo.example.com/` → `foo.example.com` |
| 2 | `path/backup-dir` | `{home}/.backup/ai-critic/foo.example.com` |
| 3 | `path/archive-extension` | `machine-backup-20260710-120000.tar.xz` |
| 4 | `schedule/enable/never-ran` | `ShouldRunOnEnable` true when never finished |
| 5 | `schedule/enable/recent-finish` | last finish 30m ago → false |
| 6 | `schedule/enable/stale-finish` | last finish 2h ago → true |
| 7 | `schedule/due/enabled-due` | enabled + due + not running → true |
| 8 | `schedule/due/running` | already running → false |
| 9 | `schedule/due/disabled` | disabled → false |
| 10 | `schedule/interval/one-hour` | interval is 3600 seconds |
| 11 | `status/off` | default off → `Status: Off` |
| 12 | `status/on-idle` | `Status: On · last 12m ago · next in 48m` |
| 13 | `status/on-running` | `Status: On · Running` |
| 14 | `status/on-error` | `Status: On · Error · 5m ago` |
| 15 | `recent/empty` | empty label `No recent backups` |
| 16 | `recent/sorted-newest-first` | paths ordered by mtime desc |
| 17 | `recent/format-entry` | `12m ago · 42 MB` |
| 18 | `retention/today-over-limit` | 12 today → keep 10 newest, delete 2 |
| 19 | `retention/past-day-multiple` | yesterday 3 → keep newest only |
| 20 | `retention/older-than-7-days` | 8 days ago → delete |
| 21 | `retention/mixed` | today + history + old apply together |
| 22 | `menu/status-children` | children = Enable, Disable only |
| 23 | `menu/when-off` | Enable active, Disable inactive |
| 24 | `menu/when-on` | Disable active, Enable inactive |
| 25 | `client/backup-menu` | remote app has Backup menu |
| 26 | `client/status-enable-disable` | Status nested Enable/Disable |
| 27 | `client/default-off` | default not auto-enabled |
| 28 | `client/stream-token-download` | stream + archive_token download path |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| server-name-from-url | path_server_name | URL https host | host only |
| backup-dir | path_backup_dir | home + serverName | `.backup/ai-critic/...` |
| archive-extension | path_archive_filename | UTC ts | `machine-backup-….tar.xz` |
| never-ran | schedule_on_enable | zero lastFinished | ShouldRun=true |
| recent-finish | schedule_on_enable | last=now-30m, interval=1h | ShouldRun=false |
| stale-finish | schedule_on_enable | last=now-2h | ShouldRun=true |
| enabled-due | schedule_due | enabled, next<=now | ShouldRun=true |
| running | schedule_due | running=true | ShouldRun=false |
| disabled | schedule_due | enabled=false | ShouldRun=false |
| one-hour | schedule_interval | — | 3600 |
| off | status_title | enabled=false / phase=off | `Status: Off` |
| on-idle | status_title | idle last/next | sealed On · last · next |
| on-running | status_title | phase=running | `Status: On · Running` |
| on-error | status_title | phase=error | `Status: On · Error · 5m ago` |
| empty | recent_empty | — | `No recent backups` |
| sorted-newest-first | recent_list | 3 entries | mtime desc paths |
| format-entry | recent_format | 12m, 42MiB | `12m ago · 42 MB` |
| today-over-limit | retention | 12 today | keep 10 / delete 2 |
| past-day-multiple | retention | 3 yesterday | keep 1 |
| older-than-7-days | retention | 8d ago | delete |
| mixed | retention | mix | all rules |
| status-children | menu_children | — | Enable, Disable |
| when-off | menu_gating | enabled=false | Enable active |
| when-on | menu_gating | enabled=true | Disable active |
| backup-menu | client | remote Swift | Backup menu |
| status-enable-disable | client | remote Swift | nested Status |
| default-off | client | remote Swift | default off |
| stream-token-download | client | remote Swift | stream+token |

## How to Run

```sh
doctest vet ./tests/macos-menubar-backup
doctest test ./tests/macos-menubar-backup/...
```

```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

// Fixed clock used by pure leaves unless a leaf overrides NowRFC3339.
const defaultNowRFC3339 = "2026-07-10T15:00:00Z"

type backupEntryDTO struct {
	Path      string `json:"path"`
	ModTime   string `json:"mod_time"` // RFC3339
	SizeBytes int64  `json:"size_bytes"`
}

type Request struct {
	Op string

	// path
	ServerURL      string
	HomeDir        string
	ServerName     string
	UTCTimeRFC3339 string

	// schedule / status shared times
	Enabled             bool
	Running             bool
	LastFinishedRFC3339 string // empty = never ran / zero time
	NextRunRFC3339      string
	NowRFC3339          string
	IntervalSeconds     int // 0 → use package default / 3600 in asserts

	// status title
	Phase     string // off | idle | running | error
	LastError string

	// recent / retention
	EntriesJSON     string // JSON array of backupEntryDTO
	EntryPath       string
	EntryModRFC3339 string
	EntrySizeBytes  int64

	// client
	ClientLeaf string
}

type Response struct {
	ServerName string
	BackupDir  string
	Filename   string

	ShouldRun       bool
	IntervalSeconds int

	StatusTitle    string
	EmptyLabel     string
	FormattedEntry string
	SortedPaths    []string

	KeepPaths   []string
	DeletePaths []string

	StatusChildren []string
	EnableActive   bool
	DisableActive  bool

	// client contract
	HasBackupMenu                bool
	HasStatusNestedEnableDisable bool
	DefaultEnabledFalse          bool
	UsesStreamTokenDownload      bool
	HasBackupNowItem             bool
	HasRevealInFinderItem        bool
	SwiftSourcesChecked          []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "path_server_name":
		resp.ServerName = menubar.ServerNameFromURL(req.ServerURL)
	case "path_backup_dir":
		resp.BackupDir = menubar.BackupDir(req.HomeDir, req.ServerName)
	case "path_archive_filename":
		utc, err := parseRFC3339(req.UTCTimeRFC3339)
		if err != nil {
			return nil, err
		}
		resp.Filename = menubar.BackupArchiveFilename(utc.UTC())
	case "schedule_on_enable":
		now, err := parseNow(req)
		if err != nil {
			return nil, err
		}
		last, err := parseOptionalTime(req.LastFinishedRFC3339)
		if err != nil {
			return nil, err
		}
		interval := intervalFromReq(req)
		resp.ShouldRun = menubar.ShouldRunOnEnable(last, now, interval)
	case "schedule_due":
		now, err := parseNow(req)
		if err != nil {
			return nil, err
		}
		next, err := parseOptionalTime(req.NextRunRFC3339)
		if err != nil {
			return nil, err
		}
		resp.ShouldRun = menubar.ShouldRunDue(req.Enabled, req.Running, next, now)
	case "schedule_interval":
		resp.IntervalSeconds = menubar.BackupIntervalSeconds
	case "status_title":
		now, err := parseNow(req)
		if err != nil {
			return nil, err
		}
		last, err := parseOptionalTime(req.LastFinishedRFC3339)
		if err != nil {
			return nil, err
		}
		next, err := parseOptionalTime(req.NextRunRFC3339)
		if err != nil {
			return nil, err
		}
		st := menubar.BackupTaskStatus{
			Enabled:        req.Enabled,
			Phase:          menubar.BackupPhase(req.Phase),
			LastFinishedAt: last,
			NextRunAt:      next,
			LastError:      req.LastError,
		}
		resp.StatusTitle = menubar.FormatBackupStatusTitle(st, now)
	case "recent_empty":
		resp.EmptyLabel = menubar.FormatBackupRecentEmptyLabel()
	case "recent_list":
		entries, err := parseEntries(req.EntriesJSON)
		if err != nil {
			return nil, err
		}
		sorted := menubar.SortBackupEntriesNewestFirst(entries)
		for _, e := range sorted {
			resp.SortedPaths = append(resp.SortedPaths, e.Path)
		}
	case "recent_format":
		now, err := parseNow(req)
		if err != nil {
			return nil, err
		}
		mod, err := parseRFC3339(req.EntryModRFC3339)
		if err != nil {
			return nil, err
		}
		entry := menubar.BackupFileEntry{
			Path:      req.EntryPath,
			ModTime:   mod,
			SizeBytes: req.EntrySizeBytes,
		}
		resp.FormattedEntry = menubar.FormatBackupEntry(entry, now)
	case "retention":
		now, err := parseNow(req)
		if err != nil {
			return nil, err
		}
		entries, err := parseEntries(req.EntriesJSON)
		if err != nil {
			return nil, err
		}
		keep, del := menubar.PruneBackupFiles(entries, now)
		for _, e := range keep {
			resp.KeepPaths = append(resp.KeepPaths, e.Path)
		}
		for _, e := range del {
			resp.DeletePaths = append(resp.DeletePaths, e.Path)
		}
	case "menu_children":
		resp.StatusChildren = menubar.BackupStatusMenuChildren()
	case "menu_gating":
		resp.EnableActive = menubar.BackupEnableItemEnabled(req.Enabled)
		resp.DisableActive = menubar.BackupDisableItemEnabled(req.Enabled)
	case "client":
		return runClientContract(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
	return resp, nil
}

func intervalFromReq(req *Request) time.Duration {
	if req.IntervalSeconds > 0 {
		return time.Duration(req.IntervalSeconds) * time.Second
	}
	return time.Duration(menubar.BackupIntervalSeconds) * time.Second
}

func parseNow(req *Request) (time.Time, error) {
	s := req.NowRFC3339
	if s == "" {
		s = defaultNowRFC3339
	}
	return parseRFC3339(s)
}

func parseRFC3339(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	return time.Parse(time.RFC3339, s)
}

func parseOptionalTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return parseRFC3339(s)
}

func parseEntries(raw string) ([]menubar.BackupFileEntry, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var dtos []backupEntryDTO
	if err := json.Unmarshal([]byte(raw), &dtos); err != nil {
		return nil, fmt.Errorf("parse EntriesJSON: %w", err)
	}
	out := make([]menubar.BackupFileEntry, 0, len(dtos))
	for _, d := range dtos {
		mt, err := parseRFC3339(d.ModTime)
		if err != nil {
			return nil, fmt.Errorf("entry %q mod_time: %w", d.Path, err)
		}
		out = append(out, menubar.BackupFileEntry{
			Path:      d.Path,
			ModTime:   mt,
			SizeBytes: d.SizeBytes,
		})
	}
	return out, nil
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
	// Include other Swift files in remote app + Shared for helpers (formatter, client).
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

	resp.HasBackupMenu = hasBackupMenu(combined)
	resp.HasStatusNestedEnableDisable = hasStatusNestedEnableDisable(combined)
	resp.DefaultEnabledFalse = hasDefaultBackupOff(combined)
	resp.UsesStreamTokenDownload = hasStreamTokenDownload(combined)
	resp.HasBackupNowItem = strings.Contains(combined, "Backup Now")
	resp.HasRevealInFinderItem = strings.Contains(combined, "Reveal in Finder")

	switch req.ClientLeaf {
	case "backup-menu",
		"status-enable-disable",
		"default-off",
		"stream-token-download":
		// fields populated above
	default:
		return nil, fmt.Errorf("unknown client leaf %q", req.ClientLeaf)
	}
	return resp, nil
}

func hasBackupMenu(src string) bool {
	if strings.Contains(src, `Menu("Backup")`) || strings.Contains(src, `Menu("Backup…")`) {
		return true
	}
	return regexp.MustCompile(`Menu\s*\(\s*"Backup`).MatchString(src) ||
		strings.Contains(src, "backup-menu") ||
		regexp.MustCompile(`(?i)accessibilityIdentifier\s*\(\s*"backup`).MatchString(src)
}

func hasStatusNestedEnableDisable(src string) bool {
	// Nested Status menu with Enable and Disable children under Backup.
	hasStatus := regexp.MustCompile(`Menu\s*\([^\)]*Status`).MatchString(src) ||
		strings.Contains(src, "Status:") ||
		strings.Contains(src, "backup-status")
	hasEnable := strings.Contains(src, `"Enable"`) || regexp.MustCompile(`Button\s*\(\s*"Enable"`).MatchString(src)
	hasDisable := strings.Contains(src, `"Disable"`) || regexp.MustCompile(`Button\s*\(\s*"Disable"`).MatchString(src)
	// Prefer structure near Backup: Enable and Disable both appear with Status.
	nearBackup := regexp.MustCompile(`(?is)Backup[\s\S]{0,1000}Enable[\s\S]{0,800}Disable|Backup[\s\S]{0,1000}Disable[\s\S]{0,800}Enable`).MatchString(src)
	return hasStatus && hasEnable && hasDisable && (nearBackup || (hasEnable && hasDisable))
}

func hasDefaultBackupOff(src string) bool {
	// Must not auto-enable periodic backup on launch.
	// Accept explicit default false / enabled = false for backup task.
	if regexp.MustCompile(`(?i)backup[A-Za-z0-9_]*[Ee]nabled\s*=\s*true`).MatchString(src) &&
		!regexp.MustCompile(`(?i)backup[A-Za-z0-9_]*[Ee]nabled\s*=\s*false`).MatchString(src) {
		// If only true assignment without false default, treat as auto-enable.
		return false
	}
	if regexp.MustCompile(`(?i)(periodicBackup|backupTask|BackupTask)[\s\S]{0,200}enabled\s*[:=]\s*false`).MatchString(src) {
		return true
	}
	if regexp.MustCompile(`(?i)enabled\s*[:=]\s*false[\s\S]{0,120}(backup|Backup)`).MatchString(src) {
		return true
	}
	// Presence of Backup menu without an on-launch enable call is OK once feature lands;
	// require either explicit false default or no force-enable on appear/init.
	forceEnable := regexp.MustCompile(`(?i)(onAppear|init\s*\(|applicationDidFinish)[\s\S]{0,400}(enableBackup|backupEnabled\s*=\s*true|setBackupEnabled\s*\(\s*true)`).MatchString(src)
	if forceEnable {
		return false
	}
	// If Backup feature code exists, count as default-off when not force-enabled.
	return hasBackupMenu(src) && !forceEnable
}

func hasStreamTokenDownload(src string) bool {
	hasStream := strings.Contains(src, "/api/remote-agent/machine/backup/stream") ||
		strings.Contains(src, "machine/backup/stream") ||
		regexp.MustCompile(`(?i)backup/stream`).MatchString(src)
	hasToken := strings.Contains(src, "archive_token") ||
		strings.Contains(src, "archiveToken") ||
		strings.Contains(src, "ArchiveToken")
	return hasStream && hasToken
}

// pathSetEqual reports whether a and b contain the same paths (order-independent).
func pathSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
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
