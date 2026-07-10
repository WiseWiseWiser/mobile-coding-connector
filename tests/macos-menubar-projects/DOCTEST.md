# macOS Menu Bar Projects Formatting Doctests

Pure-function tests for `macosapp/menubar` Projects menu formatters — the Go
spec mirrored by the Swift `ai-critic-macos` client when rendering the Projects
submenu. Optional client leaves inspect Swift sources for menu presence and
`/api/wrk/...` paths.

# DSN (Domain Specific Notion)

**Participants**

- **Project menu formatters (`macosapp/menubar`)** — pure helpers that format
  per-project submenu titles (name + branch + clean/dirty/error), worktree row
  titles (basename + clean/dirty), and the empty-registry placeholder.
- **macOS menu bar (`AICriticApp.swift`)** — top-level **Projects** menu
  alongside Services / Terminals; nested menu per project; **New Worktree…**
  (UI action; formatters cover labels only).
- **Swift `ServerClient`** — HTTP client on main server port `23712` for
  `GET /api/wrk/projects` and `POST /api/wrk/worktrees`.
- **Test harness** — invokes Go formatters with leaf inputs or inspects Swift
  sources; no UI automation, no network, no live wrkserver.

**Behaviors**

- `FormatProjectTitle(name, branch, clean, errMsg)`:
  - non-empty `errMsg` → `{name} ⚠ Error`
  - clean → `{name} ● {branch}`
  - dirty → `{name} ○ {branch}`
- `FormatWorktreeTitle(name, clean)`:
  - clean → `{name} ● Clean`
  - dirty → `{name} ○ Dirty`
- `FormatProjectsEmptyLabel()` → exact `No wrk projects`
- Swift (optional): local app exposes Projects submenu; ServerClient uses
  `/api/wrk/projects` and `/api/wrk/worktrees` on port 23712.

## Version

0.0.2

## Decision Tree

```
[macos-menubar-projects]
 |
 +-- project-title/                     (GROUP)  per-project submenu title
 |    +-- clean/                        (LEAF)   name + ● + branch
 |    +-- dirty/                        (LEAF)   name + ○ + branch
 |    +-- error/                        (LEAF)   name + ⚠ Error
 |
 +-- worktree-title/                    (GROUP)  linked worktree row title
 |    +-- clean/                        (LEAF)   basename ● Clean
 |    +-- dirty/                        (LEAF)   basename ○ Dirty
 |
 +-- empty/                             (GROUP)  empty Projects list
 |    +-- label/                        (LEAF)   `No wrk projects`
 |
 +-- client/                            (GROUP)  Swift source contracts
      +-- projects-submenu/             (LEAF)   top-level Projects menu
      +-- api-wrk-paths/                (LEAF)   ServerClient /api/wrk/...
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `project-title/clean` | `demo ● main` |
| 2 | `project-title/dirty` | `demo ○ main` |
| 3 | `project-title/error` | `demo ⚠ Error` |
| 4 | `worktree-title/clean` | `feat-login ● Clean` |
| 5 | `worktree-title/dirty` | `feat-login ○ Dirty` |
| 6 | `empty/label` | `FormatProjectsEmptyLabel` → `No wrk projects` |
| 7 | `client/projects-submenu` | Local Swift has Projects submenu |
| 8 | `client/api-wrk-paths` | ServerClient paths use `/api/wrk/...` |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| clean | project_title | name=demo, branch=main, clean=true | `demo ● main` |
| dirty | project_title | name=demo, branch=main, clean=false | `demo ○ main` |
| error | project_title | name=demo, errMsg=missing | `demo ⚠ Error` |
| clean | worktree_title | name=feat-login, clean=true | `feat-login ● Clean` |
| dirty | worktree_title | name=feat-login, clean=false | `feat-login ○ Dirty` |
| label | empty | — | `No wrk projects` |
| projects-submenu | client | AICriticApp.swift | Projects menu present |
| api-wrk-paths | client | ServerClient.swift | `/api/wrk/projects` + worktrees |

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
	Op string // "project_title" | "worktree_title" | "empty" | "client"

	// project_title
	Name   string
	Branch string
	Clean  bool
	ErrMsg string

	// worktree_title uses Name + Clean

	// client
	ClientLeaf string
}

type Response struct {
	Title      string
	EmptyLabel string

	// client contract
	HasProjectsMenu      bool
	HasAPIWrkProjects    bool
	HasAPIWrkWorktrees   bool
	ServerPort           int
	UsesServerPort23712  bool
	SwiftSourcesChecked  []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "project_title":
		resp.Title = menubar.FormatProjectTitle(req.Name, req.Branch, req.Clean, req.ErrMsg)
	case "worktree_title":
		resp.Title = menubar.FormatWorktreeTitle(req.Name, req.Clean)
	case "empty":
		resp.EmptyLabel = menubar.FormatProjectsEmptyLabel()
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
	resp.SwiftSourcesChecked = []string{appPath, serverClientPath}

	appSrc, err := os.ReadFile(appPath)
	if err != nil {
		return nil, fmt.Errorf("read AICriticApp.swift: %w", err)
	}
	serverSrc, err := os.ReadFile(serverClientPath)
	if err != nil {
		return nil, fmt.Errorf("read ServerClient.swift: %w", err)
	}
	appStr := string(appSrc)
	serverStr := string(serverSrc)
	combined := appStr + "\n" + serverStr

	resp.ServerPort = config.DefaultServerPort
	resp.UsesServerPort23712 = strings.Contains(serverStr, "23712") || resp.ServerPort == 23712

	switch req.ClientLeaf {
	case "projects-submenu":
		// Top-level Projects menu (Menu/label/string containing Projects).
		resp.HasProjectsMenu = strings.Contains(appStr, "Projects") &&
			(strings.Contains(appStr, "Menu") || strings.Contains(appStr, "menu"))
	case "api-wrk-paths":
		resp.HasAPIWrkProjects = strings.Contains(combined, "/api/wrk/projects")
		resp.HasAPIWrkWorktrees = strings.Contains(combined, "/api/wrk/worktrees")
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
