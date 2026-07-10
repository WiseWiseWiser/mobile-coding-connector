# wrkserver HTTP Handlers Doctests

HTTP types + handlers for listing wrk projects and creating worktrees under
`wrkcli/wrkserver`. Host mounts via `Register(mux, base)` — leaf paths are fixed;
base prefix is host-owned (ai-critic uses `/api/wrk`).

# DSN (Domain Specific Notion)

**Participants**

- **wrkserver (`wrkcli/wrkserver`)** — owns HTTP DTOs, `ListProjects` /
  `CreateWorktree` handlers, and `Register(mux, base)` mounting
  `GET {base}/projects` and `POST {base}/worktrees`.
- **wrk storage (`$WRK_HOME`)** — `projects.json` registry and default
  `worktrees/` create root; injectable via `Options.WrkHome` for tests.
- **git worktrees** — main repo status + linked worktree list (`git worktree
  list` linked semantics); create under `$WRK_HOME/worktrees` with wrk naming.
- **Host mux (ai-critic)** — passes base (e.g. `/api/wrk`); auth middleware is
  host-owned and out of scope for wrkserver.
- **Test harness** — temp `WRK_HOME` + temp git repos; invokes handlers via
  `httptest` or `Register` + mux; no `wrk` binary.

**Behaviors**

- `GET {base}/projects` → `200` + `{"projects":[...]}` envelope (never bare array).
- Each `ProjectStatus`: path, name (`filepath.Base`), branch/commit/subject when
  available, `clean`, optional `error`, and `worktrees` (linked only; main status
  lives on the project fields).
- Missing recorded path → project still present with non-empty `error`.
- `POST {base}/worktrees` with `project_path` (required) and optional `task`.
  Empty/omit/whitespace-only `task` → no task slug. Non-empty task → slug
  (e.g. `"Fix Login"` → `fix-login`) in path and branch.
- Create path under `$WRK_HOME/worktrees/`; naming matches wrk defaults
  (`basename-branchToken-date[-slug][-N]`, branch `branchBase-date[-slug][-N]`).
- Validation errors → 4xx JSON `{"error":"..."}`.
- `Register` joins base (no trailing slash required) with fixed leaves; base is
  not hardcoded inside the package.

## Version

0.0.2

## Decision Tree

```
[wrkserver]
 |
 +-- list/                              (GROUP)  GET projects
 |    +-- empty-registry/               (LEAF)   no projects → {"projects":[]}
 |    +-- clean-main-no-worktrees/      (LEAF)   one clean main, worktrees empty
 |    +-- main-with-linked/             (LEAF)   linked WTs + clean flags
 |    +-- missing-path/                 (LEAF)   recorded path missing → error
 |
 +-- create/                            (GROUP)  POST worktrees
 |    +-- no-task/                      (LEAF)   omit task → no slug
 |    +-- with-task/                    (LEAF)   "Fix Login" → fix-login slug
 |    +-- whitespace-task/              (LEAF)   whitespace task → no slug
 |    +-- missing-project-path/         (LEAF)   4xx + {"error":...}
 |    +-- non-git-path/                 (LEAF)   4xx + error for non-git path
 |
 +-- register/                          (GROUP)  Register(mux, base) routing
      +-- api-wrk-get-projects/         (LEAF)   GET /api/wrk/projects
      +-- api-wrk-post-worktrees/       (LEAF)   POST /api/wrk/worktrees
      +-- custom-base-get-projects/     (LEAF)   GET /custom/projects (host base)
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `list/empty-registry` | Empty registry → 200, `{"projects":[]}` |
| 2 | `list/clean-main-no-worktrees` | One clean main, no linked worktrees |
| 3 | `list/main-with-linked` | Main + linked worktrees; dirty flag on linked |
| 4 | `list/missing-path` | Recorded path missing on disk → non-empty `error` |
| 5 | `create/no-task` | Create without task → path under worktrees/, no task slug |
| 6 | `create/with-task` | Task `"Fix Login"` → path/branch include `fix-login` |
| 7 | `create/whitespace-task` | Whitespace-only task → same as no task |
| 8 | `create/missing-project-path` | Missing `project_path` → 4xx + `{"error":...}` |
| 9 | `create/non-git-path` | Non-git `project_path` → 4xx + error |
| 10a | `register/api-wrk-get-projects` | `Register(..., "/api/wrk")` → GET `/api/wrk/projects` |
| 10b | `register/api-wrk-post-worktrees` | `Register(..., "/api/wrk")` → POST `/api/wrk/worktrees` |
| 11 | `register/custom-base-get-projects` | `Register(..., "/custom")` → GET `/custom/projects` |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| empty-registry | list | empty WrkHome | 200, projects=[] |
| clean-main-no-worktrees | list | one clean git main | clean=true, worktrees empty |
| main-with-linked | list | main + dirty linked WT | worktrees listed, clean flags |
| missing-path | list | path not on disk | project.error non-empty |
| no-task | create | omit task | path under worktrees/, no fix-login |
| with-task | create | task=Fix Login | path+branch contain fix-login |
| whitespace-task | create | task=`"   "` | no slug (like no-task) |
| missing-project-path | create | no project_path | 4xx + error body |
| non-git-path | create | plain dir | 4xx + error body |
| api-wrk-get-projects | register | base=/api/wrk | GET works (not 404) |
| api-wrk-post-worktrees | register | base=/api/wrk | POST works (not 404) |
| custom-base-get-projects | register | base=/custom | GET /custom/projects works |

## How to Run

```sh
doctest vet ./tests/wrkserver
doctest test ./tests/wrkserver/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/dot-pkgs/go-pkgs/wrkcli/wrkserver"
)

// Request drives one wrkserver HTTP scenario.
type Request struct {
	Op string // "list" | "create" | "register"

	// Isolated wrk home (required for all ops that touch storage).
	WrkHome string

	// create
	ProjectPath string
	Task        string
	OmitTask    bool // when true, JSON body omits "task" entirely

	// register
	Base   string // e.g. "/api/wrk" or "/custom"
	Method string // HTTP method after Register
	Path   string // full request path after Register
	// Optional create-shaped body when Method=POST under register
	RegisterBody string
}

// Response captures HTTP status + parsed JSON fields used by asserts.
type Response struct {
	StatusCode  int
	Body        string
	ContentType string

	// list
	Projects []map[string]any

	// create success
	Path   string
	Branch string

	// error envelope
	Error string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	if req.WrkHome == "" {
		return nil, fmt.Errorf("WrkHome is required")
	}
	srv := wrkserver.New(wrkserver.Options{WrkHome: req.WrkHome})

	switch req.Op {
	case "list":
		return serveHandler(t, http.MethodGet, "/projects", nil, srv.ListProjects)
	case "create":
		body, err := buildCreateBody(req)
		if err != nil {
			return nil, err
		}
		return serveHandler(t, http.MethodPost, "/worktrees", body, srv.CreateWorktree)
	case "register":
		if req.Base == "" || req.Path == "" || req.Method == "" {
			return nil, fmt.Errorf("register requires Base, Method, Path")
		}
		mux := http.NewServeMux()
		srv.Register(mux, req.Base)
		var body io.Reader
		if req.RegisterBody != "" {
			body = strings.NewReader(req.RegisterBody)
		} else if req.Method == http.MethodPost {
			// default empty object so handler can return validation error (not 404)
			body = strings.NewReader(`{}`)
		}
		httpReq := httptest.NewRequest(req.Method, req.Path, body)
		if body != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httpReq)
		return parseResponse(rr), nil
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func serveHandler(t *testing.T, method, path string, body []byte, h http.HandlerFunc) (*Response, error) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	httpReq := httptest.NewRequest(method, path, rdr)
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httpReq)
	return parseResponse(rr), nil
}

func buildCreateBody(req *Request) ([]byte, error) {
	m := map[string]any{}
	if req.ProjectPath != "" {
		m["project_path"] = req.ProjectPath
	}
	if !req.OmitTask {
		m["task"] = req.Task
	}
	return json.Marshal(m)
}

func parseResponse(rr *httptest.ResponseRecorder) *Response {
	resp := &Response{
		StatusCode:  rr.Code,
		Body:        rr.Body.String(),
		ContentType: rr.Header().Get("Content-Type"),
	}
	var generic map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &generic); err != nil {
		return resp
	}
	if errMsg, ok := generic["error"].(string); ok {
		resp.Error = errMsg
	}
	if path, ok := generic["path"].(string); ok {
		resp.Path = path
	}
	if branch, ok := generic["branch"].(string); ok {
		resp.Branch = branch
	}
	if raw, ok := generic["projects"]; ok {
		if arr, ok := raw.([]any); ok {
			// Distinguish missing key (nil) from present empty array.
			resp.Projects = make([]map[string]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					resp.Projects = append(resp.Projects, m)
				}
			}
		}
	}
	return resp
}

// --- fixture helpers used by SETUP chain ---

func mkTempDir(t *testing.T, pattern string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

// writeProjectsJSON writes projects.json under wrkHome.
func writeProjectsJSON(t *testing.T, wrkHome string, paths []string) {
	t.Helper()
	if err := os.MkdirAll(wrkHome, 0o755); err != nil {
		t.Fatalf("mkdir wrk home: %v", err)
	}
	type project struct {
		Path    string `json:"path"`
		AddedAt string `json:"added_at"`
		Source  string `json:"source"`
	}
	type file struct {
		Version  int       `json:"version"`
		Projects []project `json:"projects"`
	}
	pf := file{Version: 1}
	for _, p := range paths {
		pf.Projects = append(pf.Projects, project{
			Path:    p,
			AddedAt: "2026-01-01T00:00:00Z",
			Source:  "manual",
		})
	}
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		t.Fatalf("marshal projects: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wrkHome, "projects.json"), data, 0o644); err != nil {
		t.Fatalf("write projects.json: %v", err)
	}
}

func gitInitWithMain(t *testing.T, dir string) {
	t.Helper()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test User")
	gitRun(t, dir, "branch", "-M", "main")
}

func gitInitialCommit(t *testing.T, dir, message string) {
	t.Helper()
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	gitRun(t, dir, "add", "README.md")
	gitRun(t, dir, "commit", "-m", message)
}

func mkCleanMainRepo(t *testing.T) string {
	t.Helper()
	dir := mkTempDir(t, "wrkserver-main-*")
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	return dir
}
```
