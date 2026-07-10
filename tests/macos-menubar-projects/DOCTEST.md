# macOS Menu Bar Projects — loading, title parts, stale-while-revalidate

Pure-function and Swift source-contract tests for the local macOS **Projects**
menu: left/right title alignment (`Leading` / `Trailing`), loading and failure
labels, and stale-while-revalidate list state while `/api/wrk/projects` is in
flight. Formatters and optional list-state reducer live in Go
(`macosapp/menubar`); Swift mirrors contracts in `ProjectsMenuFormatter` and
`AICriticApp`.

# DSN (Domain Specific Notion)

**Participants**

- **Project / worktree title-parts formatters (`macosapp/menubar`)** — pure
  helpers that split each row into **Leading** (basename only) and **Trailing**
  decoration (`● branch`, `○ branch`, `⚠ Error`, `● Clean`, `○ Dirty`). Legacy
  single-string `FormatProjectTitle` / `FormatWorktreeTitle` compose as
  `Leading + "  " + Trailing` when a single string is still needed.
- **List status labels** — pure strings for the disabled placeholder row when
  there are no project submenus: `Loading…`, `No wrk projects`,
  `Failed to load projects`.
- **List status selector** — pure helper that picks which placeholder applies
  given `(loading, projectCount, errMsg)` when count is zero (or returns empty
  when rows are present).
- **Projects list state reducer** — pure transitions for
  `ProjectsListState{Projects, Loading, Error}`: start refresh (do not clear
  items), success (replace items, clear error), failure (keep items, set error).
- **macOS menu bar (`AICriticApp.swift`)** — top-level **Projects** menu;
  renders titles with left/right alignment (`HStack` + `Spacer`); tracks
  `projectsLoading` / optional load error without wiping the last good list.
- **Swift `ServerClient`** — `GET /api/wrk/projects` and worktree create on
  port `23712`.
- **Test harness** — calls Go helpers with leaf inputs or inspects Swift
  sources; no UI automation, no network, no live wrkserver.

**Behaviors**

- `FormatProjectTitleParts(name, branch, clean, errMsg)`:
  - non-empty `errMsg` → Leading=`name`, Trailing=`⚠ Error`
  - clean → Leading=`name`, Trailing=`● {branch}`
  - dirty → Leading=`name`, Trailing=`○ {branch}`
- `FormatWorktreeTitleParts(name, clean)`:
  - clean → Leading=`name`, Trailing=`● Clean`
  - dirty → Leading=`name`, Trailing=`○ Dirty`
- Legacy: `FormatProjectTitle` / `FormatWorktreeTitle` → `Leading + "  " + Trailing`
- `FormatProjectsLoadingLabel()` → exact `Loading…` (unicode ellipsis)
- `FormatProjectsEmptyLabel()` → exact `No wrk projects`
- `FormatProjectsLoadFailedLabel()` → exact `Failed to load projects`
- `FormatProjectsListStatusLabel(loading, count, err)`:
  - count > 0 → `""` (show project menus; optional updating cue is out of band)
  - loading && count == 0 → `Loading…`
  - !loading && count == 0 && err != "" → `Failed to load projects`
  - !loading && count == 0 && err == "" → `No wrk projects`
- `ApplyProjectsRefreshStart(s)` → Loading=true; Projects and Error unchanged
- `ApplyProjectsRefreshSuccess(s, list)` → Projects=list; Error=""; Loading=false
- `ApplyProjectsRefreshFailure(s, err)` → Projects unchanged; Error=err; Loading=false
- Swift: title rows use parts + `HStack { Text(leading); Spacer(); Text(trailing) }`
  (or equivalent left/right layout); `AppState.projectsLoading`; do not clear
  `projects` on refresh start or failure; show `Loading…` when loading and empty.

## Version

0.0.2

## Decision Tree

```
[macos-menubar-projects]
 |
 +-- project-title/                     (GROUP)  project Leading/Trailing
 |    +-- clean/                        (LEAF)   demo / ● main
 |    +-- dirty/                        (LEAF)   demo / ○ main
 |    +-- error/                        (LEAF)   demo / ⚠ Error
 |
 +-- worktree-title/                    (GROUP)  worktree Leading/Trailing
 |    +-- clean/                        (LEAF)   feat-login / ● Clean
 |    +-- dirty/                        (LEAF)   feat-login / ○ Dirty
 |
 +-- labels/                            (GROUP)  placeholder string constants
 |    +-- empty/                        (LEAF)   No wrk projects
 |    +-- loading/                      (LEAF)   Loading…
 |    +-- failed/                       (LEAF)   Failed to load projects
 |
 +-- list-status/                       (GROUP)  empty-area label selection
 |    +-- loading-empty/                (LEAF)   loading+empty → Loading…
 |    +-- idle-empty/                   (LEAF)   idle empty → No wrk projects
 |    +-- failed-empty/                 (LEAF)   error empty → Failed…
 |    +-- has-rows/                     (LEAF)   count>0 → ""
 |
 +-- refresh-state/                     (GROUP)  stale-while-revalidate reducer
 |    +-- start-keeps-items/            (LEAF)   start: loading, items kept
 |    +-- fail-keeps-items/             (LEAF)   fail: items kept, error set
 |    +-- success-replaces/             (LEAF)   success: replace, clear error
 |
 +-- client/                            (GROUP)  Swift source contracts
      +-- projects-submenu/             (LEAF)   top-level Projects menu
      +-- api-wrk-paths/                (LEAF)   ServerClient /api/wrk/...
      +-- title-parts-hstack/           (LEAF)   parts + HStack left/right
      +-- projects-loading-flag/        (LEAF)   projectsLoading; no clear
      +-- loading-when-empty/           (LEAF)   Loading… when load && empty
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `project-title/clean` | Leading `demo`, Trailing `● main`; legacy composed |
| 2 | `project-title/dirty` | Leading `demo`, Trailing `○ main`; legacy composed |
| 3 | `project-title/error` | Leading `demo`, Trailing `⚠ Error`; legacy composed |
| 4 | `worktree-title/clean` | Leading `feat-login`, Trailing `● Clean` |
| 5 | `worktree-title/dirty` | Leading `feat-login`, Trailing `○ Dirty` |
| 6 | `labels/empty` | `FormatProjectsEmptyLabel` → `No wrk projects` |
| 7 | `labels/loading` | `FormatProjectsLoadingLabel` → `Loading…` |
| 8 | `labels/failed` | `FormatProjectsLoadFailedLabel` → `Failed to load projects` |
| 9 | `list-status/loading-empty` | empty+loading → Loading… (not empty registry) |
| 10 | `list-status/idle-empty` | idle empty → No wrk projects |
| 11 | `list-status/failed-empty` | empty+error → Failed to load projects |
| 12 | `list-status/has-rows` | count>0 → no status placeholder |
| 13 | `refresh-state/start-keeps-items` | start keeps prior projects |
| 14 | `refresh-state/fail-keeps-items` | fail keeps prior; sets error |
| 15 | `refresh-state/success-replaces` | success replaces list; clears error |
| 16 | `client/projects-submenu` | Local Swift has Projects submenu |
| 17 | `client/api-wrk-paths` | ServerClient paths use `/api/wrk/...` |
| 18 | `client/title-parts-hstack` | Project/worktree titles use parts + HStack |
| 19 | `client/projects-loading-flag` | `projectsLoading`; list not cleared on start/fail |
| 20 | `client/loading-when-empty` | Shows Loading… when loading and empty |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| clean | project_title | name=demo, branch=main, clean | L=`demo` T=`● main` |
| dirty | project_title | name=demo, branch=main, !clean | L=`demo` T=`○ main` |
| error | project_title | name=demo, errMsg set | L=`demo` T=`⚠ Error` |
| clean | worktree_title | name=feat-login, clean | L=`feat-login` T=`● Clean` |
| dirty | worktree_title | name=feat-login, !clean | L=`feat-login` T=`○ Dirty` |
| empty | label | kind=empty | `No wrk projects` |
| loading | label | kind=loading | `Loading…` |
| failed | label | kind=failed | `Failed to load projects` |
| loading-empty | list_status | loading, count=0 | `Loading…` |
| idle-empty | list_status | !loading, count=0 | `No wrk projects` |
| failed-empty | list_status | !loading, count=0, err | `Failed to load projects` |
| has-rows | list_status | count=2 | `""` |
| start-keeps-items | refresh_state | action=start, prior=[a] | loading, [a] |
| fail-keeps-items | refresh_state | action=failure, prior=[a] | !loading, [a], err |
| success-replaces | refresh_state | action=success, new=[b] | !loading, [b], err="" |
| projects-submenu | client | AICriticApp.swift | Projects menu |
| api-wrk-paths | client | ServerClient.swift | `/api/wrk/...` |
| title-parts-hstack | client | Swift sources | HStack + parts |
| projects-loading-flag | client | AppState | projectsLoading |
| loading-when-empty | client | menu body | Loading… path |

## How to Run

```sh
doctest vet ./tests/macos-menubar-projects
doctest test ./tests/macos-menubar-projects/...
```

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
	"github.com/xhd2015/ai-critic/server/config"
)

type Request struct {
	// Op selects Run branch:
	// "project_title" | "worktree_title" | "label" | "list_status" |
	// "refresh_state" | "client"
	Op string

	// project_title / worktree_title
	Name   string
	Branch string
	Clean  bool
	ErrMsg string

	// label
	LabelKind string // "empty" | "loading" | "failed"

	// list_status
	Loading      bool
	ProjectCount int
	// ErrMsg reused for load-error text when non-empty

	// refresh_state
	PriorProjects []string
	PriorLoading  bool
	PriorError    string
	RefreshAction string // "start" | "success" | "failure"
	NewProjects   []string
	FailError     string

	// client
	ClientLeaf string
}

type Response struct {
	// title parts + legacy single string
	Leading  string
	Trailing string
	Title    string

	// labels / list status
	Label string

	// refresh-state snapshot
	Projects []string
	Loading  bool
	Error    string

	// client contracts
	HasProjectsMenu             bool
	HasAPIWrkProjects           bool
	HasAPIWrkWorktrees          bool
	ServerPort                  int
	UsesServerPort23712         bool
	UsesTitlePartsHStack        bool
	HasProjectsLoadingFlag      bool
	KeepsProjectsOnRefreshStart bool
	KeepsProjectsOnRefreshFail  bool
	ShowsLoadingWhenEmpty       bool
	SwiftSourcesChecked         []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "project_title":
		parts := menubar.FormatProjectTitleParts(req.Name, req.Branch, req.Clean, req.ErrMsg)
		resp.Leading = parts.Leading
		resp.Trailing = parts.Trailing
		resp.Title = menubar.FormatProjectTitle(req.Name, req.Branch, req.Clean, req.ErrMsg)
	case "worktree_title":
		parts := menubar.FormatWorktreeTitleParts(req.Name, req.Clean)
		resp.Leading = parts.Leading
		resp.Trailing = parts.Trailing
		resp.Title = menubar.FormatWorktreeTitle(req.Name, req.Clean)
	case "label":
		switch req.LabelKind {
		case "empty":
			resp.Label = menubar.FormatProjectsEmptyLabel()
		case "loading":
			resp.Label = menubar.FormatProjectsLoadingLabel()
		case "failed":
			resp.Label = menubar.FormatProjectsLoadFailedLabel()
		default:
			return nil, fmt.Errorf("unknown LabelKind %q", req.LabelKind)
		}
	case "list_status":
		resp.Label = menubar.FormatProjectsListStatusLabel(req.Loading, req.ProjectCount, req.ErrMsg)
	case "refresh_state":
		s := menubar.ProjectsListState{
			Projects: append([]string(nil), req.PriorProjects...),
			Loading:  req.PriorLoading,
			Error:    req.PriorError,
		}
		switch req.RefreshAction {
		case "start":
			s = menubar.ApplyProjectsRefreshStart(s)
		case "success":
			s = menubar.ApplyProjectsRefreshSuccess(s, req.NewProjects)
		case "failure":
			s = menubar.ApplyProjectsRefreshFailure(s, req.FailError)
		default:
			return nil, fmt.Errorf("unknown RefreshAction %q", req.RefreshAction)
		}
		resp.Projects = s.Projects
		resp.Loading = s.Loading
		resp.Error = s.Error
	case "client":
		return runClientContract(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
	return resp, nil
}

func runClientContract(t *testing.T, req *Request, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}
	appPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "AICriticApp.swift")
	serverClientPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "ServerClient.swift")
	formatterPath := filepath.Join(moduleRoot, "macos-ai-critic", "Shared", "ProjectsMenuFormatter.swift")
	resp.SwiftSourcesChecked = []string{appPath, serverClientPath, formatterPath}

	appSrc, err := os.ReadFile(appPath)
	if err != nil {
		return nil, fmt.Errorf("read AICriticApp.swift: %w", err)
	}
	serverSrc, err := os.ReadFile(serverClientPath)
	if err != nil {
		return nil, fmt.Errorf("read ServerClient.swift: %w", err)
	}
	formatterSrc, err := os.ReadFile(formatterPath)
	if err != nil {
		// Formatter may live next to app; tolerate missing for older layouts by using empty.
		formatterSrc = nil
	}
	appStr := string(appSrc)
	serverStr := string(serverSrc)
	formatterStr := string(formatterSrc)
	combined := appStr + "\n" + serverStr + "\n" + formatterStr

	resp.ServerPort = config.DefaultServerPort
	resp.UsesServerPort23712 = strings.Contains(serverStr, "23712") || resp.ServerPort == 23712

	switch req.ClientLeaf {
	case "projects-submenu":
		resp.HasProjectsMenu = strings.Contains(appStr, "Projects") &&
			(strings.Contains(appStr, "Menu") || strings.Contains(appStr, "menu"))
	case "api-wrk-paths":
		resp.HasAPIWrkProjects = strings.Contains(combined, "/api/wrk/projects")
		resp.HasAPIWrkWorktrees = strings.Contains(combined, "/api/wrk/worktrees")
	case "title-parts-hstack":
		// Contract: title parts API + HStack/Spacer (or Label with left/right).
		hasParts := strings.Contains(combined, "formatProjectTitleParts") ||
			strings.Contains(combined, "ProjectTitleParts") ||
			(strings.Contains(combined, "leading") && strings.Contains(combined, "trailing")) ||
			(strings.Contains(combined, "Leading") && strings.Contains(combined, "Trailing"))
		hasHStack := strings.Contains(appStr, "HStack") && strings.Contains(appStr, "Spacer")
		resp.UsesTitlePartsHStack = hasParts && hasHStack
	case "projects-loading-flag":
		resp.HasProjectsLoadingFlag = strings.Contains(appStr, "projectsLoading")
		// Stale-while-revalidate: refresh must not assign projects = [] on start/fail.
		// Heuristic: projectsLoading present and no clear-on-error wipe of projects.
		resp.KeepsProjectsOnRefreshStart = resp.HasProjectsLoadingFlag &&
			!strings.Contains(appStr, "projects = []")
		resp.KeepsProjectsOnRefreshFail = resp.KeepsProjectsOnRefreshStart
	case "loading-when-empty":
		hasLoadingLabel := strings.Contains(combined, "Loading…") ||
			strings.Contains(combined, "formatProjectsLoadingLabel") ||
			strings.Contains(combined, "FormatProjectsLoadingLabel")
		usesLoadingGate := strings.Contains(appStr, "projectsLoading") &&
			(strings.Contains(appStr, "isEmpty") || strings.Contains(appStr, ".isEmpty"))
		resp.ShowsLoadingWhenEmpty = hasLoadingLabel && usesLoadingGate
	default:
		return nil, fmt.Errorf("unknown client leaf %q", req.ClientLeaf)
	}
	return resp, nil
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
