# macOS Menu Bar Cron Formatting Doctests

Pure-function tests for Cron menu helpers (`macosapp/menubar`) and request
builders (`macosapp/cronapi`), plus Swift source contracts for local
(`ai-critic-macos`) and remote (`ai-critic-remote-macos`) menu-bar apps.

Mirror of Services submenu pattern: nested per-task menus, pure Go formatters
as the Swift contract, pure path builders for `/api/cron-tasks*` and log SSE.

# DSN (Domain Specific Notion)

**Participants**

- **Cron menu formatters (`macosapp/menubar`)** — pure helpers that map
  `CronTaskStatus` fields to submenu titles (name + status glyph + short
  schedule), empty/not-configured placeholders, Run Now gating, Enable vs
  Disable action choice, and enable/disable alert copy (server message or
  client fallback).
- **Cron API builders (`macosapp/cronapi`)** — pure path/request builders for
  `GET /api/cron-tasks` and `POST /api/cron-tasks/{run|enable|disable}?id=`,
  with optional Bearer auth; mirror `macosapp/serviceapi` for Services.
- **Local macOS menu bar (`ai-critic-macos`)** — top-level `Menu("Cron")` after
  Services and before Terminals; nested per-task menus; local `ServerClient`
  for list/run/enable/disable; View Logs via SSE; accessibility id `cron-menu`.
- **Remote macOS menu bar (`ai-critic-remote-macos`)** — same Cron UX against
  configured remote base URL + Bearer (`ServiceClient` or shared client); shows
  `Not configured` when endpoint missing; never uses keep-alive daemon port for
  cron APIs.
- **Log stream window** — View Logs (local + remote) streams
  `GET /api/logs/stream?path=…&lines=…` using task `logPath` (SSE with auth on
  remote).
- **Test harness** — invokes Go helpers with leaf inputs or inspects Swift
  sources; no UI automation, no live network, no server process.

**Behaviors**

- `FormatCronTaskTitle(name, status, enabled, scheduleMode, interval, cronExpr)`:
  - `running` → `{name} ● Running · {sched}`
  - `idle` + enabled → `{name} ○ Idle · {sched}`
  - `idle` + disabled → `{name} ○ Idle (disabled) · {sched}`
  - `error` → `{name} ⚠ Error · {sched}`
  - other/unknown + enabled → `{name} ○ {status} · {sched}` (raw status; empty
    status treated as `Idle`)
  - other/unknown + disabled → `{name} ○ {status} (disabled) · {sched}`
  - schedule suffix: `interval` → `every {interval}`; `cron` → `cron {cronExpr}`
    (UTC expr as returned by API).
- `FormatCronTasksEmptyLabel()`: exact `No cron tasks configured`.
- `FormatCronNotConfiguredLabel()`: exact `Not configured` (remote missing
  endpoint; same as Services/Terminals).
- `CanRunCronTask(status)`: false when `status == "running"`; true otherwise
  (idle, error, unknown).
- `ShowEnableCronAction(enabled)` / reuse enable-toggle: disabled task shows
  **Enable**; enabled shows **Disable**.
- `CronToggleAlertMessage(serverMessage)`: non-empty trimmed server message
  wins; else fallback `Task updated`.
- `cronapi`: list path `/api/cron-tasks`; actions
  `/api/cron-tasks/{run|enable|disable}?id=`; Bearer when token set; omit
  Authorization when token empty; reject empty base or empty id.
- Swift: both apps expose `Menu("Cron")` with accessibility id `cron-menu`,
  nested per-task menus (Run Now / Enable|Disable / View Logs… / History…
  disabled), placement after Services before Terminals; View Logs uses
  `/api/logs/stream`; remote uses configured base URL + auth; cron included in
  30s periodic refresh and top-level Refresh path.

## Version

0.0.2

## Decision Tree

```
[macos-menubar-cron]
 |
 +-- title/                              (GROUP)  per-task submenu title strings
 |    +-- running-interval/              (LEAF)   `backup ● Running · every 5m`
 |    +-- idle-interval/                 (LEAF)   `backup ○ Idle · every 5m`
 |    +-- idle-disabled-cron/            (LEAF)   `nightly ○ Idle (disabled) · cron 0 1 * * *`
 |    +-- error-interval/                (LEAF)   `scrape ⚠ Error · every 1m`
 |    +-- idle-cron/                     (LEAF)   `nightly ○ Idle · cron 0 1 * * *`
 |
 +-- action/                             (GROUP)  Run Now + Enable/Disable gating
 |    +-- run-when-running/              (LEAF)   CanRun=false when status=running
 |    +-- run-when-idle/                 (LEAF)   CanRun=true when status=idle
 |    +-- run-when-error/                (LEAF)   CanRun=true when status=error
 |    +-- show-enable/                   (LEAF)   enabled=false → Enable
 |    +-- show-disable/                  (LEAF)   enabled=true → Disable (not Enable)
 |
 +-- empty/                              (GROUP)  empty list + remote missing endpoint
 |    +-- label/                         (LEAF)   `No cron tasks configured`
 |    +-- not-configured/                (LEAF)   `Not configured`
 |
 +-- alert/                              (GROUP)  enable/disable NSAlert copy
 |    +-- prefer-server-message/         (LEAF)   non-empty server message wins
 |    +-- empty-uses-fallback/           (LEAF)   empty/whitespace → `Task updated`
 |
 +-- cronapi/                            (GROUP)  pure path + request builders
 |    +-- list-path/                     (LEAF)   GET path `/api/cron-tasks`
 |    +-- run-path/                      (LEAF)   POST `/api/cron-tasks/run?id=`
 |    +-- enable-path/                   (LEAF)   POST `/api/cron-tasks/enable?id=`
 |    +-- disable-path/                  (LEAF)   POST `/api/cron-tasks/disable?id=`
 |    +-- list-with-auth/                (LEAF)   Bearer when token set
 |    +-- list-no-auth/                  (LEAF)   no Authorization when token empty
 |    +-- action-requires-id/            (LEAF)   empty id → error
 |
 +-- client/                             (GROUP)  Swift source contracts (both apps)
      +-- local-cron-menu/               (LEAF)   local Menu("Cron") + cron-menu id
      +-- remote-cron-menu/              (LEAF)   remote Menu("Cron") + cron-menu id
      +-- nested-task-actions/           (LEAF)   nested Menu; Run/Enable/Disable/Logs/History
      +-- history-disabled/              (LEAF)   History… disabled placeholder
      +-- view-logs-sse/                 (LEAF)   /api/logs/stream (not local tail)
      +-- remote-not-configured/         (LEAF)   remote shows Not configured path
      +-- remote-cron-api-base/          (LEAF)   base URL + Bearer; not keep-alive port
      +-- menu-placement/                (LEAF)   after Services, before Terminals
      +-- periodic-refresh/              (LEAF)   cron included in 30s refresh path
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `title/running-interval` | running + interval → `backup ● Running · every 5m` |
| 2 | `title/idle-interval` | idle+enabled + interval → `backup ○ Idle · every 5m` |
| 3 | `title/idle-disabled-cron` | idle+disabled + cron → `nightly ○ Idle (disabled) · cron 0 1 * * *` |
| 4 | `title/error-interval` | error + interval → `scrape ⚠ Error · every 1m` |
| 5 | `title/idle-cron` | idle+enabled + cron → `nightly ○ Idle · cron 0 1 * * *` |
| 6 | `action/run-when-running` | `CanRunCronTask("running")` → false |
| 7 | `action/run-when-idle` | `CanRunCronTask("idle")` → true |
| 8 | `action/run-when-error` | `CanRunCronTask("error")` → true |
| 9 | `action/show-enable` | disabled → ShowEnable=true |
| 10 | `action/show-disable` | enabled → ShowEnable=false |
| 11 | `empty/label` | empty list label → `No cron tasks configured` |
| 12 | `empty/not-configured` | remote missing endpoint → `Not configured` |
| 13 | `alert/prefer-server-message` | server message preferred over fallback |
| 14 | `alert/empty-uses-fallback` | empty server message → `Task updated` |
| 15 | `cronapi/list-path` | list path is `/api/cron-tasks` |
| 16 | `cronapi/run-path` | run path encodes id |
| 17 | `cronapi/enable-path` | enable path encodes id |
| 18 | `cronapi/disable-path` | disable path encodes id |
| 19 | `cronapi/list-with-auth` | list request includes `Bearer` token |
| 20 | `cronapi/list-no-auth` | empty token omits Authorization |
| 21 | `cronapi/action-requires-id` | empty id returns error |
| 22 | `client/local-cron-menu` | local app has Cron menu + accessibility id |
| 23 | `client/remote-cron-menu` | remote app has Cron menu + accessibility id |
| 24 | `client/nested-task-actions` | nested per-task actions present |
| 25 | `client/history-disabled` | History… disabled placeholder |
| 26 | `client/view-logs-sse` | View Logs uses `/api/logs/stream` SSE |
| 27 | `client/remote-not-configured` | remote Cron empty/not-configured path |
| 28 | `client/remote-cron-api-base` | remote cron APIs use base URL + auth |
| 29 | `client/menu-placement` | Cron after Services, before Terminals |
| 30 | `client/periodic-refresh` | refresh path includes cron tasks |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| running-interval | title | name=backup, status=running, enabled=true, mode=interval, interval=5m | Title exact |
| idle-interval | title | name=backup, status=idle, enabled=true, mode=interval, interval=5m | Title exact |
| idle-disabled-cron | title | name=nightly, status=idle, enabled=false, mode=cron, cronExpr=`0 1 * * *` | Title exact |
| error-interval | title | name=scrape, status=error, enabled=true, mode=interval, interval=1m | Title exact |
| idle-cron | title | name=nightly, status=idle, enabled=true, mode=cron, cronExpr=`0 1 * * *` | Title exact |
| run-when-running | action | status=running | CanRun=false |
| run-when-idle | action | status=idle | CanRun=true |
| run-when-error | action | status=error | CanRun=true |
| show-enable | action | enabled=false | ShowEnable=true |
| show-disable | action | enabled=true | ShowEnable=false |
| label | empty | — | EmptyLabel=`No cron tasks configured` |
| not-configured | empty | NotConfigured=true | EmptyLabel=`Not configured` |
| prefer-server-message | alert | serverMsg set | AlertMessage=serverMsg |
| empty-uses-fallback | alert | serverMsg empty | AlertMessage=`Task updated` |
| list-path | cronapi | leaf=list-path | Path=`/api/cron-tasks` |
| run-path | cronapi | action=run, id=task-1 | Path contains run + id |
| enable-path | cronapi | action=enable, id=task-1 | Path contains enable + id |
| disable-path | cronapi | action=disable, id=task-1 | Path contains disable + id |
| list-with-auth | cronapi | token=secret | Auth=`Bearer secret` |
| list-no-auth | cronapi | token="" | Auth omitted |
| action-requires-id | cronapi | id="" | Err non-nil |
| local-cron-menu | client | local Swift | HasLocalCronMenu |
| remote-cron-menu | client | remote Swift | HasRemoteCronMenu |
| nested-task-actions | client | both apps | nested Run/Enable/Disable/Logs/History |
| history-disabled | client | both apps | History disabled |
| view-logs-sse | client | both + Shared | `/api/logs/stream` |
| remote-not-configured | client | remote Swift | Not configured in Cron |
| remote-cron-api-base | client | remote + Shared | cron path + base/auth, not 23312 |
| menu-placement | client | both apps | Services → Cron → Terminals |
| periodic-refresh | client | both apps | cron in refresh loop |

## How to Run

```sh
doctest vet ./tests/macos-menubar-cron
doctest test ./tests/macos-menubar-cron/...
```

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/cronapi"
	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

type Request struct {
	Op string

	// title / action
	Name         string
	Status       string
	Enabled      bool
	ScheduleMode string
	Interval     string
	CronExpr     string

	// empty
	NotConfigured bool

	// alert
	ServerMessage string

	// cronapi
	CronAPILeaf string
	BaseURL     string
	Token       string
	TaskID      string
	CronAction  string

	// client source contracts
	ClientLeaf string
}

type Response struct {
	Title      string
	CanRun     bool
	ShowEnable bool
	EmptyLabel string
	AlertMessage string

	// cronapi
	Method      string
	Path        string
	URL         string
	AuthHeader  string
	HasAuth     bool
	BuildErr    string
	BuildOK     bool

	// client contract flags
	HasLocalCronMenu          bool
	HasRemoteCronMenu         bool
	LocalAccessibilityID      bool
	RemoteAccessibilityID     bool
	HasNestedTaskMenu         bool
	HasRunNow                 bool
	HasEnableDisable          bool
	HasViewLogs               bool
	HasHistoryDisabled        bool
	HasLogStreamEndpoint      bool
	ViewLogsUsesStream        bool
	RemoteShowsNotConfigured  bool
	RemoteCronUsesBaseURL     bool
	RemoteCronUsesAuth        bool
	RemoteAvoidsKeepAlivePort bool
	MenuPlacementOK           bool
	HasPeriodicCronRefresh    bool
	SwiftSourcesChecked       []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "title":
		resp.Title = menubar.FormatCronTaskTitle(
			req.Name, req.Status, req.Enabled,
			req.ScheduleMode, req.Interval, req.CronExpr,
		)
	case "action":
		resp.CanRun = menubar.CanRunCronTask(req.Status)
		resp.ShowEnable = menubar.ShowEnableCronAction(req.Enabled)
	case "empty":
		if req.NotConfigured {
			resp.EmptyLabel = menubar.FormatCronNotConfiguredLabel()
		} else {
			resp.EmptyLabel = menubar.FormatCronTasksEmptyLabel()
		}
	case "alert":
		resp.AlertMessage = menubar.CronToggleAlertMessage(req.ServerMessage)
	case "cronapi":
		return runCronAPI(t, req, resp)
	case "client":
		return runClientContract(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
	return resp, nil
}

func runCronAPI(t *testing.T, req *Request, resp *Response) (*Response, error) {
	switch req.CronAPILeaf {
	case "list-path":
		resp.Path = cronapi.ListCronTasksPath()
		resp.Method = "GET"
	case "run-path", "enable-path", "disable-path":
		action := cronapi.CronAction(req.CronAction)
		resp.Path = cronapi.CronActionPath(action, req.TaskID)
		resp.Method = "POST"
	case "list-with-auth", "list-no-auth":
		built, err := cronapi.BuildListCronTasksRequest(req.BaseURL, req.Token)
		if err != nil {
			resp.BuildErr = err.Error()
			resp.BuildOK = false
			return resp, nil
		}
		resp.BuildOK = true
		resp.Method = built.Method
		resp.URL = built.URL
		if h, ok := built.Headers["Authorization"]; ok {
			resp.HasAuth = true
			resp.AuthHeader = h
		}
	case "action-requires-id":
		_, err := cronapi.BuildCronActionRequest(req.BaseURL, req.Token, cronapi.ActionRun, req.TaskID)
		if err != nil {
			resp.BuildErr = err.Error()
			resp.BuildOK = false
		} else {
			resp.BuildOK = true
		}
	default:
		return nil, fmt.Errorf("unknown cronapi leaf %q", req.CronAPILeaf)
	}
	return resp, nil
}

func runClientContract(t *testing.T, req *Request, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}
	localApp := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "AICriticApp.swift")
	remoteApp := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-remote-macos", "AICriticApp.swift")
	localServerClient := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "ServerClient.swift")
	localLogTail := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "LogTailWindow.swift")
	sharedDir := filepath.Join(moduleRoot, "macos-ai-critic", "Shared")

	localSrc, err := os.ReadFile(localApp)
	if err != nil {
		return nil, fmt.Errorf("read local AICriticApp.swift: %w", err)
	}
	remoteSrc, err := os.ReadFile(remoteApp)
	if err != nil {
		return nil, fmt.Errorf("read remote AICriticApp.swift: %w", err)
	}
	localStr := string(localSrc)
	remoteStr := string(remoteSrc)

	serverClientStr := ""
	if b, e := os.ReadFile(localServerClient); e == nil {
		serverClientStr = string(b)
		resp.SwiftSourcesChecked = append(resp.SwiftSourcesChecked, localServerClient)
	}
	logTailStr := ""
	if b, e := os.ReadFile(localLogTail); e == nil {
		logTailStr = string(b)
		resp.SwiftSourcesChecked = append(resp.SwiftSourcesChecked, localLogTail)
	}

	sharedCombined := ""
	_ = filepath.Walk(sharedDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil || info.IsDir() {
			return walkErr
		}
		if !strings.HasSuffix(path, ".swift") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		sharedCombined += "\n" + string(b)
		resp.SwiftSourcesChecked = append(resp.SwiftSourcesChecked, path)
		return nil
	})
	resp.SwiftSourcesChecked = append([]string{localApp, remoteApp}, resp.SwiftSourcesChecked...)

	both := localStr + "\n" + remoteStr + "\n" + sharedCombined
	allSrc := both + "\n" + serverClientStr + "\n" + logTailStr

	resp.HasLocalCronMenu = hasCronMenu(localStr)
	resp.HasRemoteCronMenu = hasCronMenu(remoteStr)
	resp.LocalAccessibilityID = hasCronAccessibilityID(localStr)
	resp.RemoteAccessibilityID = hasCronAccessibilityID(remoteStr)

	resp.HasNestedTaskMenu = hasNestedCronTaskMenu(localStr) || hasNestedCronTaskMenu(remoteStr) ||
		hasNestedCronTaskMenu(sharedCombined)
	resp.HasRunNow = strings.Contains(both, "Run Now")
	resp.HasEnableDisable = (strings.Contains(both, `Button("Enable")`) || strings.Contains(both, "Enable")) &&
		(strings.Contains(both, `Button("Disable")`) || strings.Contains(both, "Disable"))
	resp.HasViewLogs = strings.Contains(both, "View Logs")
	resp.HasHistoryDisabled = hasHistoryDisabled(both)

	resp.HasLogStreamEndpoint = strings.Contains(allSrc, "/api/logs/stream")
	resp.ViewLogsUsesStream = resp.HasLogStreamEndpoint &&
		!regexp.MustCompile(`View Logs[\s\S]{0,200}/usr/bin/tail`).MatchString(allSrc)

	resp.RemoteShowsNotConfigured = hasCronNotConfigured(remoteStr)
	resp.RemoteCronUsesBaseURL = hasRemoteCronAPI(remoteStr + "\n" + sharedCombined)
	resp.RemoteCronUsesAuth = strings.Contains(remoteStr+"\n"+sharedCombined, "Authorization") ||
		strings.Contains(remoteStr+"\n"+sharedCombined, "Bearer") ||
		strings.Contains(remoteStr+"\n"+sharedCombined, "token")
	// Cron must not target keep-alive daemon port 23312 for business APIs
	resp.RemoteAvoidsKeepAlivePort = !regexp.MustCompile(`23312[\s\S]{0,80}cron-tasks|cron-tasks[\s\S]{0,80}23312`).MatchString(allSrc)

	resp.MenuPlacementOK = hasCronMenuPlacement(localStr) && hasCronMenuPlacement(remoteStr)
	resp.HasPeriodicCronRefresh = hasPeriodicCronRefresh(localStr) || hasPeriodicCronRefresh(remoteStr) ||
		hasPeriodicCronRefresh(sharedCombined)

	switch req.ClientLeaf {
	case "local-cron-menu",
		"remote-cron-menu",
		"nested-task-actions",
		"history-disabled",
		"view-logs-sse",
		"remote-not-configured",
		"remote-cron-api-base",
		"menu-placement",
		"periodic-refresh":
		// fields populated above
	default:
		return nil, fmt.Errorf("unknown client leaf %q", req.ClientLeaf)
	}
	return resp, nil
}

func hasCronMenu(src string) bool {
	if strings.Contains(src, `Menu("Cron")`) {
		return true
	}
	return regexp.MustCompile(`Menu\s*\(\s*"Cron"`).MatchString(src)
}

func hasCronAccessibilityID(src string) bool {
	return strings.Contains(src, `cron-menu`) ||
		regexp.MustCompile(`accessibilityIdentifier\(\s*"cron-menu"`).MatchString(src)
}

func hasNestedCronTaskMenu(src string) bool {
	// Nested Menu inside Cron for per-task actions.
	// Go RE2 caps quantifier max at 1000 — keep bounds ≤1000.
	if regexp.MustCompile(`Menu\s*\(\s*"Cron"[\s\S]{0,1000}ForEach[\s\S]{0,800}Menu\s*\(`).MatchString(src) {
		return true
	}
	if regexp.MustCompile(`Menu\s*\(\s*"Cron"[\s\S]{0,1000}Menu\s*\(`).MatchString(src) &&
		(strings.Contains(src, "Run Now") || strings.Contains(src, "cronTasks") || strings.Contains(src, "CronTask")) {
		return true
	}
	return strings.Contains(src, "CronMenuFormatter") && strings.Contains(src, "Run Now")
}

func hasHistoryDisabled(src string) bool {
	if !strings.Contains(src, "History") {
		return false
	}
	// History… with .disabled(true) nearby, or disabled History button
	return regexp.MustCompile(`History[^\n]{0,40}[\s\S]{0,120}\.disabled\(\s*true\s*\)`).MatchString(src) ||
		regexp.MustCompile(`\.disabled\(\s*true\s*\)[\s\S]{0,120}History`).MatchString(src) ||
		(strings.Contains(src, "History…") && strings.Contains(src, ".disabled(true)")) ||
		(strings.Contains(src, `Button("History`) && strings.Contains(src, "disabled"))
}

func hasCronNotConfigured(src string) bool {
	// Cron menu shows Not configured when remote endpoint missing.
	// Go RE2 caps quantifier max at 1000 — keep bounds ≤1000.
	if regexp.MustCompile(`Menu\s*\(\s*"Cron"[\s\S]{0,1000}Not configured`).MatchString(src) {
		return true
	}
	return strings.Contains(src, "FormatCronNotConfiguredLabel") ||
		strings.Contains(src, "formatCronNotConfiguredLabel") ||
		(strings.Contains(src, "Not configured") &&
			(strings.Contains(src, "cronTasks") || strings.Contains(src, "CronTask") || hasCronMenu(src)))
}

func hasRemoteCronAPI(src string) bool {
	return strings.Contains(src, "/api/cron-tasks") ||
		strings.Contains(src, "listCronTasks") ||
		strings.Contains(src, "ListCronTasks") ||
		strings.Contains(src, "cronTasksPath") ||
		regexp.MustCompile(`(?i)cron.?tasks`).MatchString(src) && strings.Contains(src, "baseURL")
}

func hasCronMenuPlacement(src string) bool {
	// Services appears before Cron, Cron before Terminals in source order
	si := strings.Index(src, `Menu("Services")`)
	if si < 0 {
		si = indexRegexp(src, `Menu\s*\(\s*"Services"`)
	}
	ci := strings.Index(src, `Menu("Cron")`)
	if ci < 0 {
		ci = indexRegexp(src, `Menu\s*\(\s*"Cron"`)
	}
	ti := strings.Index(src, `Menu("Terminals")`)
	if ti < 0 {
		ti = indexRegexp(src, `Menu\s*\(\s*"Terminals"`)
	}
	return si >= 0 && ci >= 0 && ti >= 0 && si < ci && ci < ti
}

func hasPeriodicCronRefresh(src string) bool {
	hasSleepOrTimer := strings.Contains(src, "Task.sleep") ||
		strings.Contains(src, "Timer") ||
		strings.Contains(src, "nanoseconds:") ||
		regexp.MustCompile(`startRefresh|refreshLoop|periodicRefresh`).MatchString(src)
	hasCronFetch := regexp.MustCompile(`(?i)(refreshCron|listCron|cron-tasks|cronTasks|CronTask)`).MatchString(src)
	// Top-level Refresh that also pulls cron
	hasRefreshButtonCron := strings.Contains(src, `Button("Refresh")`) && hasCronFetch
	return (hasSleepOrTimer && hasCronFetch) || hasRefreshButtonCron
}

func indexRegexp(src, pattern string) int {
	loc := regexp.MustCompile(pattern).FindStringIndex(src)
	if loc == nil {
		return -1
	}
	return loc[0]
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
