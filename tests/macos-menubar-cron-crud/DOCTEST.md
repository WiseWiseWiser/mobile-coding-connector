# macOS Menu Bar Cron CRUD Doctests

Pure-function tests for Cron create / update / delete request builders
(`macosapp/cronapi`), delete gating + confirm copy (`macosapp/menubar`),
local↔UTC cron conversion helpers (`macosapp/cronapi`), and Swift source
contracts for local (`ai-critic-macos`) and remote (`ai-critic-remote-macos`)
menu-bar apps — Cron Editor window (Option A) + menu actions.

Style: pure Go helpers + Swift source contracts; no UI automation, no live
network, no server process. Sibling of operate-only `tests/macos-menubar-cron/`.

# DSN (Domain Specific Notion)

**Participants**

- **Cron API builders (`macosapp/cronapi`)** — pure path/request builders for
  create (`POST /api/cron-tasks` + JSON body), update (`PUT /api/cron-tasks` +
  JSON body with `id`), and delete (`DELETE /api/cron-tasks?id=`), with optional
  Bearer auth; mirror existing list/run/enable/disable builders. Body is
  `CronTaskDef` (name, command, optional workingDir, scheduleMode, interval /
  cronExpr, timeout, enabled). **`cronExpr` as stored/sent is always UTC.**
  UI does not send `extraEnv`.
- **Delete / editor helpers (`macosapp/menubar`)** — `CanDeleteCronTask(status)`
  (false only when `status == "running"`); `FormatDeleteCronConfirm(name)` →
  `Delete cron task "{name}"?`.
- **Local↔UTC convert (`macosapp/cronapi`)** — pure helpers aligned with CLI
  (`cmd/agentcli/cron.go`): safe simple local 5-field expr + fixed-offset zone
  → UTC on save; unsafe ranges/lists/steps/DST → error. On edit open:
  safe UTC→local for display; unsafe → keep stored UTC (UI shows UTC
  indication and pass-through on save).
- **Local macOS menu bar (`ai-critic-macos`)** — `Menu("Cron")` gains
  per-task `Edit…` / `Delete…` (Delete disabled when running) and bottom
  `New Cron Task…`; opens shared Cron Editor window; `ServerClient`
  create/update/delete.
- **Remote macOS menu bar (`ai-critic-remote-macos`)** — same CRUD UX via
  `ServiceClient` base URL + Bearer; `New Cron Task…` disabled when remote not
  configured.
- **Cron Editor (shared Swift UI)** — form: name, command, optional working
  dir, schedule (Interval | Cron), timeout (default `1h`, must be > 0),
  enabled (default on). Save → POST (create) or PUT (update with id) →
  refresh → close; Cancel → close; errors → alert.
- **Test harness** — invokes Go helpers with leaf inputs or inspects Swift
  sources; no UI automation.

**Behaviors**

- `BuildCreateCronTaskRequest(base, token, def)` → `POST {base}/api/cron-tasks`,
  JSON body, optional Bearer; rejects empty base; body includes name/command
  and schedule fields; `cronExpr` is UTC when present.
- `BuildUpdateCronTaskRequest(base, token, def)` → `PUT {base}/api/cron-tasks`,
  JSON body with non-empty `id`; rejects empty id.
- `BuildDeleteCronTaskRequest(base, token, id)` →
  `DELETE {base}/api/cron-tasks?id=…`; rejects empty id.
- Path helpers: create/update path `/api/cron-tasks`; delete path
  `/api/cron-tasks?id=` (id URL-encoded).
- `CanDeleteCronTask(status)`: false when `status == "running"`; true for
  idle, error, unknown.
- `FormatDeleteCronConfirm(name)`: exact `Delete cron task "{name}"?`.
- `ConvertLocalCronToUTC(expr, loc)`: safe simple + fixed offset → UTC 5-field;
  unsafe → error.
- `ConvertUTCCronToLocal(expr, loc)`: reverse when safe; unsafe → error
  (caller shows stored UTC).
- Swift: both apps — `New Cron Task…` at bottom of Cron menu (empty list still
  shows empty label + divider + New); per-task `Edit…` and `Delete…` after
  operate actions separator; Delete disabled when running; Cron Editor Save
  wired to create or update; models expose definition fields for editor
  prefill (name, command, workingDir, scheduleMode, interval, cronExpr,
  timeout, enabled).

## Version

0.0.2

## Decision Tree

```
[macos-menubar-cron-crud]
 |
 +-- cronapi/                            (GROUP)  create/update/delete request builders
 |    +-- create-path/                   (LEAF)   POST /api/cron-tasks + JSON body
 |    +-- update-path/                   (LEAF)   PUT /api/cron-tasks + JSON body with id
 |    +-- delete-path/                   (LEAF)   DELETE /api/cron-tasks?id=
 |    +-- delete-requires-id/            (LEAF)   empty id → error
 |
 +-- delete-gate/                        (GROUP)  CanDelete + confirm dialog copy
 |    +-- when-running/                  (LEAF)   CanDelete=false when status=running
 |    +-- when-idle/                     (LEAF)   CanDelete=true when status=idle
 |    +-- when-error/                    (LEAF)   CanDelete=true when status=error
 |    +-- confirm-message/               (LEAF)   FormatDeleteCronConfirm → exact string
 |
 +-- convert/                            (GROUP)  local↔UTC cron expression helpers
 |    +-- local-to-utc-safe/             (LEAF)   fixed-offset + simple → UTC
 |    +-- local-to-utc-unsafe/           (LEAF)   ranges/lists → error
 |    +-- utc-to-local-safe/             (LEAF)   reverse convert for edit open
 |
 +-- client/                             (GROUP)  Swift source contracts (both apps)
      +-- local-new-cron-task/           (LEAF)   local has New Cron Task…
      +-- remote-new-cron-task/          (LEAF)   remote has New Cron Task…
      +-- new-at-bottom/                 (LEAF)   New Cron Task… at bottom of Cron menu
      +-- per-task-edit/                 (LEAF)   per-task Edit…
      +-- per-task-delete/               (LEAF)   per-task Delete…
      +-- delete-disabled-running/       (LEAF)   Delete disabled when running
      +-- editor-save-create/            (LEAF)   Editor Save → create / POST
      +-- editor-save-update/            (LEAF)   Editor Save → update / PUT
      +-- definition-fields/             (LEAF)   CronTaskDefinition / editor fields
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `cronapi/create-path` | POST `/api/cron-tasks` with JSON body (name, command, schedule, UTC cronExpr) |
| 2 | `cronapi/update-path` | PUT `/api/cron-tasks` with JSON body including id |
| 3 | `cronapi/delete-path` | DELETE `/api/cron-tasks?id=` encodes id |
| 4 | `cronapi/delete-requires-id` | empty id → build error |
| 5 | `delete-gate/when-running` | `CanDeleteCronTask("running")` → false |
| 6 | `delete-gate/when-idle` | `CanDeleteCronTask("idle")` → true |
| 7 | `delete-gate/when-error` | `CanDeleteCronTask("error")` → true |
| 8 | `delete-gate/confirm-message` | confirm copy `Delete cron task "backup"?` |
| 9 | `convert/local-to-utc-safe` | `0 9 * * *` @ Etc/GMT-8 → `0 1 * * *` |
| 10 | `convert/local-to-utc-unsafe` | range/list expr → error |
| 11 | `convert/utc-to-local-safe` | `0 1 * * *` @ Etc/GMT-8 → `0 9 * * *` |
| 12 | `client/local-new-cron-task` | local app has `New Cron Task…` |
| 13 | `client/remote-new-cron-task` | remote app has `New Cron Task…` |
| 14 | `client/new-at-bottom` | New Cron Task… is last item in Cron menu |
| 15 | `client/per-task-edit` | nested menu has `Edit…` |
| 16 | `client/per-task-delete` | nested menu has `Delete…` |
| 17 | `client/delete-disabled-running` | Delete gated by running / canDelete |
| 18 | `client/editor-save-create` | Editor Save wired to create/POST |
| 19 | `client/editor-save-update` | Editor Save wired to update/PUT |
| 20 | `client/definition-fields` | definition/editor fields present for prefill |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| create-path | cronapi | name=backup, command=echo, mode=cron, cronExpr=`0 1 * * *`, timeout=1h | Method=POST, path `/api/cron-tasks`, body JSON, BuildOK |
| update-path | cronapi | id=task-1 + same fields | Method=PUT, id in body, BuildOK |
| delete-path | cronapi | id=task-1 | Method=DELETE, path has id |
| delete-requires-id | cronapi | id="" | BuildOK=false, BuildErr non-empty |
| when-running | delete-gate | status=running | CanDelete=false |
| when-idle | delete-gate | status=idle | CanDelete=true |
| when-error | delete-gate | status=error | CanDelete=true |
| confirm-message | delete-gate | name=backup | Confirm=`Delete cron task "backup"?` |
| local-to-utc-safe | convert | expr=`0 9 * * *`, loc=Etc/GMT-8 | UTC=`0 1 * * *`, ConvertOK |
| local-to-utc-unsafe | convert | expr=`0 9-17 * * 1-5` | ConvertOK=false |
| utc-to-local-safe | convert | expr=`0 1 * * *`, loc=Etc/GMT-8 | local=`0 9 * * *` |
| local-new-cron-task | client | local Swift | HasLocalNewCronTask |
| remote-new-cron-task | client | remote Swift | HasRemoteNewCronTask |
| new-at-bottom | client | both apps | NewCronTaskAtBottom |
| per-task-edit | client | both apps | HasPerTaskEdit |
| per-task-delete | client | both apps | HasPerTaskDelete |
| delete-disabled-running | client | both apps | DeleteDisabledWhenRunning |
| editor-save-create | client | both + Shared | EditorSaveCreates |
| editor-save-update | client | both + Shared | EditorSaveUpdates |
| definition-fields | client | Shared models | HasDefinitionFields |

## How to Run

```sh
doctest vet ./tests/macos-menubar-cron-crud
doctest test -count=1 ./tests/macos-menubar-cron-crud/...
```

```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/macosapp/cronapi"
	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

type Request struct {
	Op string

	// delete-gate
	Status string
	Name   string

	// cronapi
	CronAPILeaf  string
	BaseURL      string
	Token        string
	TaskID       string
	Command      string
	WorkingDir   string
	ScheduleMode string
	Interval     string
	CronExpr     string
	Timeout      string
	Enabled      *bool

	// convert
	ConvertLeaf string
	LocalExpr   string
	UTCExpr     string
	TZName      string

	// client source contracts
	ClientLeaf string
}

type Response struct {
	// delete-gate
	CanDelete      bool
	ConfirmMessage string

	// cronapi
	Method     string
	Path       string
	URL        string
	AuthHeader string
	HasAuth    bool
	Body       string
	BuildErr   string
	BuildOK    bool
	// body field probes
	BodyHasName    bool
	BodyHasCommand bool
	BodyHasID      bool
	BodyCronExpr   string
	BodyMethodOK   bool

	// convert
	ConvertedExpr string
	ConvertOK     bool
	ConvertErr    string

	// client contract flags
	HasLocalNewCronTask       bool
	HasRemoteNewCronTask      bool
	NewCronTaskAtBottom       bool
	HasPerTaskEdit            bool
	HasPerTaskDelete          bool
	DeleteDisabledWhenRunning bool
	EditorSaveCreates         bool
	EditorSaveUpdates         bool
	HasDefinitionFields       bool
	HasCronEditor             bool
	SwiftSourcesChecked       []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "delete-gate":
		return runDeleteGate(t, req, resp)
	case "cronapi":
		return runCronAPI(t, req, resp)
	case "convert":
		return runConvert(t, req, resp)
	case "client":
		return runClientContract(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runDeleteGate(t *testing.T, req *Request, resp *Response) (*Response, error) {
	_ = t
	// when-running / when-idle / when-error set Status; confirm-message sets Name.
	resp.CanDelete = menubar.CanDeleteCronTask(req.Status)
	if req.Name != "" {
		resp.ConfirmMessage = menubar.FormatDeleteCronConfirm(req.Name)
	}
	return resp, nil
}

func runCronAPI(t *testing.T, req *Request, resp *Response) (*Response, error) {
	_ = t
	def := cronapi.CronTaskDef{
		ID:           req.TaskID,
		Name:         req.Name,
		Command:      req.Command,
		WorkingDir:   req.WorkingDir,
		ScheduleMode: req.ScheduleMode,
		Interval:     req.Interval,
		CronExpr:     req.CronExpr,
		Timeout:      req.Timeout,
		Enabled:      req.Enabled,
	}
	switch req.CronAPILeaf {
	case "create-path":
		resp.Path = cronapi.CreateCronTasksPath()
		built, err := cronapi.BuildCreateCronTaskRequest(req.BaseURL, req.Token, def)
		if err != nil {
			resp.BuildErr = err.Error()
			resp.BuildOK = false
			resp.Method = "POST"
			return resp, nil
		}
		fillBuilt(resp, built)
		resp.BodyMethodOK = resp.Method == "POST"
		probeBody(resp, built.Body)
	case "update-path":
		resp.Path = cronapi.UpdateCronTasksPath()
		built, err := cronapi.BuildUpdateCronTaskRequest(req.BaseURL, req.Token, def)
		if err != nil {
			resp.BuildErr = err.Error()
			resp.BuildOK = false
			resp.Method = "PUT"
			return resp, nil
		}
		fillBuilt(resp, built)
		resp.BodyMethodOK = resp.Method == "PUT"
		probeBody(resp, built.Body)
	case "delete-path":
		resp.Path = cronapi.DeleteCronTaskPath(req.TaskID)
		built, err := cronapi.BuildDeleteCronTaskRequest(req.BaseURL, req.Token, req.TaskID)
		if err != nil {
			resp.BuildErr = err.Error()
			resp.BuildOK = false
			resp.Method = "DELETE"
			return resp, nil
		}
		fillBuilt(resp, built)
		resp.BodyMethodOK = resp.Method == "DELETE"
	case "delete-requires-id":
		_, err := cronapi.BuildDeleteCronTaskRequest(req.BaseURL, req.Token, req.TaskID)
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

func fillBuilt(resp *Response, built cronapi.CronRequest) {
	resp.BuildOK = true
	resp.Method = built.Method
	resp.URL = built.URL
	resp.Body = string(built.Body)
	if h, ok := built.Headers["Authorization"]; ok {
		resp.HasAuth = true
		resp.AuthHeader = h
	}
	// Derive path from URL when base present
	if i := strings.Index(built.URL, "/api/"); i >= 0 {
		resp.Path = built.URL[i:]
	}
}

func probeBody(resp *Response, body []byte) {
	if len(body) == 0 {
		return
	}
	resp.Body = string(body)
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return
	}
	if v, ok := m["name"].(string); ok && v != "" {
		resp.BodyHasName = true
	}
	if v, ok := m["command"].(string); ok && v != "" {
		resp.BodyHasCommand = true
	}
	if v, ok := m["id"].(string); ok && v != "" {
		resp.BodyHasID = true
	}
	if v, ok := m["cronExpr"].(string); ok {
		resp.BodyCronExpr = v
	}
}

func runConvert(t *testing.T, req *Request, resp *Response) (*Response, error) {
	_ = t
	loc, err := time.LoadLocation(req.TZName)
	if err != nil {
		// Fallback for environments without zoneinfo: fixed UTC+8.
		if req.TZName == "Etc/GMT-8" {
			loc = time.FixedZone("UTC+8", 8*3600)
		} else {
			return nil, fmt.Errorf("load location %q: %w", req.TZName, err)
		}
	}
	switch req.ConvertLeaf {
	case "local-to-utc-safe", "local-to-utc-unsafe":
		out, err := cronapi.ConvertLocalCronToUTC(req.LocalExpr, loc)
		if err != nil {
			resp.ConvertOK = false
			resp.ConvertErr = err.Error()
			return resp, nil
		}
		resp.ConvertOK = true
		resp.ConvertedExpr = out
	case "utc-to-local-safe":
		out, err := cronapi.ConvertUTCCronToLocal(req.UTCExpr, loc)
		if err != nil {
			resp.ConvertOK = false
			resp.ConvertErr = err.Error()
			return resp, nil
		}
		resp.ConvertOK = true
		resp.ConvertedExpr = out
	default:
		return nil, fmt.Errorf("unknown convert leaf %q", req.ConvertLeaf)
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
	allSrc := both + "\n" + serverClientStr

	resp.HasLocalNewCronTask = hasNewCronTask(localStr) || hasNewCronTask(sharedCombined)
	resp.HasRemoteNewCronTask = hasNewCronTask(remoteStr) || hasNewCronTask(sharedCombined)
	resp.NewCronTaskAtBottom = hasNewCronTaskAtBottom(localStr) || hasNewCronTaskAtBottom(remoteStr) ||
		hasNewCronTaskAtBottom(sharedCombined)
	resp.HasPerTaskEdit = hasPerTaskEdit(both)
	resp.HasPerTaskDelete = hasPerTaskDelete(both)
	resp.DeleteDisabledWhenRunning = hasDeleteDisabledWhenRunning(both)
	resp.HasCronEditor = hasCronEditor(allSrc)
	resp.EditorSaveCreates = hasEditorSaveCreate(allSrc)
	resp.EditorSaveUpdates = hasEditorSaveUpdate(allSrc)
	resp.HasDefinitionFields = hasDefinitionFields(allSrc)

	switch req.ClientLeaf {
	case "local-new-cron-task",
		"remote-new-cron-task",
		"new-at-bottom",
		"per-task-edit",
		"per-task-delete",
		"delete-disabled-running",
		"editor-save-create",
		"editor-save-update",
		"definition-fields":
		// fields populated above
	default:
		return nil, fmt.Errorf("unknown client leaf %q", req.ClientLeaf)
	}
	return resp, nil
}

func hasNewCronTask(src string) bool {
	return strings.Contains(src, "New Cron Task") ||
		strings.Contains(src, "New Cron Task…") ||
		strings.Contains(src, "New Cron Task...")
}

func hasNewCronTaskAtBottom(src string) bool {
	// Within Menu("Cron") ... end, New Cron Task appears after Divider and near the end.
	// Go RE2 caps quantifier max at 1000.
	if !hasNewCronTask(src) {
		return false
	}
	// Prefer: Divider then New Cron Task within Cron menu span
	if regexp.MustCompile(`Menu\s*\(\s*"Cron"[\s\S]{0,1000}Divider\(\)[\s\S]{0,400}New Cron Task`).MatchString(src) {
		return true
	}
	// Or: empty label / ForEach then New Cron Task as last button in Cron menu
	return regexp.MustCompile(`Menu\s*\(\s*"Cron"[\s\S]{0,1000}New Cron Task[\s\S]{0,200}\}`).MatchString(src) ||
		(strings.Contains(src, "New Cron Task") && strings.Contains(src, "Divider()"))
}

func hasPerTaskEdit(src string) bool {
	return strings.Contains(src, `Button("Edit…")`) ||
		strings.Contains(src, `Button("Edit...")`) ||
		strings.Contains(src, `"Edit…"`) ||
		regexp.MustCompile(`Button\s*\(\s*"Edit`).MatchString(src)
}

func hasPerTaskDelete(src string) bool {
	return strings.Contains(src, `Button("Delete…")`) ||
		strings.Contains(src, `Button("Delete...")`) ||
		strings.Contains(src, `"Delete…"`) ||
		regexp.MustCompile(`Button\s*\(\s*"Delete`).MatchString(src)
}

func hasDeleteDisabledWhenRunning(src string) bool {
	// Delete button .disabled(...) with canDelete / running / CanDeleteCronTask
	if regexp.MustCompile(`Delete[\s\S]{0,200}\.disabled\(`).MatchString(src) {
		return true
	}
	return strings.Contains(src, "canDeleteCronTask") ||
		strings.Contains(src, "CanDeleteCronTask") ||
		strings.Contains(src, "canDelete")
}

func hasCronEditor(src string) bool {
	return strings.Contains(src, "CronEditor") ||
		strings.Contains(src, "CronEditorView") ||
		strings.Contains(src, "Cron Editor") ||
		regexp.MustCompile(`(?i)cron.?editor`).MatchString(src)
}

func hasEditorSaveCreate(src string) bool {
	// Save path calls createCronTask / createCron / POST create
	hasCreate := strings.Contains(src, "createCronTask") ||
		strings.Contains(src, "CreateCronTask") ||
		strings.Contains(src, "createCron") ||
		regexp.MustCompile(`(?i)create.*cron|cron.*create`).MatchString(src) &&
			(strings.Contains(src, "POST") || strings.Contains(src, "MethodPost") || strings.Contains(src, "httpMethod"))
	// Editor context: CronEditor or isNew / editing id nil
	hasEditor := hasCronEditor(src) || strings.Contains(src, "isNew") || strings.Contains(src, "New Cron Task")
	return hasCreate && hasEditor
}

func hasEditorSaveUpdate(src string) bool {
	hasUpdate := strings.Contains(src, "updateCronTask") ||
		strings.Contains(src, "UpdateCronTask") ||
		strings.Contains(src, "updateCron") ||
		(regexp.MustCompile(`(?i)update.*cron|cron.*update`).MatchString(src) &&
			(strings.Contains(src, "PUT") || strings.Contains(src, "MethodPut") || strings.Contains(src, "httpMethod")))
	hasEditor := hasCronEditor(src) || strings.Contains(src, "Edit…") || strings.Contains(src, "Edit...")
	return hasUpdate && hasEditor
}

func hasDefinitionFields(src string) bool {
	// CronTaskDefinition or Codable fields used by editor
	needles := []string{"workingDir", "scheduleMode", "cronExpr", "timeout", "command"}
	hits := 0
	for _, n := range needles {
		if strings.Contains(src, n) {
			hits++
		}
	}
	hasType := strings.Contains(src, "CronTaskDefinition") ||
		strings.Contains(src, "struct CronTask") ||
		hasCronEditor(src)
	return hasType && hits >= 3
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
