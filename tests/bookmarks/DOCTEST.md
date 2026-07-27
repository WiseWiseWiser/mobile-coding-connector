# Bookmarks Management Doctests

Classic TDD contract for **local-agent bookmarks**: Chrome-style folder/url tree
stored in `~/.ai-critic/bookmarks.json`, HTTP API on the local server, CLI
`local-agent bookmarks …`, pure browser resolve helper, and menu-bar label
helpers. No production `server/bookmarks` package exists yet — leaves are
expected **RED** until the implementer lands code.

Most leaves are **L2 in-process**: `bookmarks.NewManagerAt` + optional
`RegisterAPIWithManager` on `httptest`, and/or `agentcli.Run(LocalProfile())`
with testhooks home overrides. No real browser launch; `open` is dry-run /
resolve-only.

# DSN (Domain Specific Notion)

**Participants**

- **bookmarks.Manager (`server/bookmarks`)** — owns `bookmarks.json` under an
  isolated home/path; load/save tree; add / update / delete / move nodes.
- **Document / Node** — versioned root document with `roots[]`; each node is
  `type=folder|url`, stable `id`, non-empty `name`; url nodes have absolute
  `url` and optional `browser`; folders have `children`.
- **ResolveBrowser** — pure helper: bookmark browser if set, else caller
  global default, else `"default"`.
- **HTTP API** — `GET/POST /api/bookmarks`, `PATCH/DELETE /api/bookmarks?id=`,
  `POST /api/bookmarks/move`; same auth pattern as other local APIs in tests
  (Bearer on mux wrapper).
- **local-agent CLI (`bookmarks`)** — list / add / add-folder / set / delete /
  move / open; help at every level; human tables + `--json`.
- **Menu label helpers** — empty submenu copy and URL item title (Go pure
  functions mirrored by Swift menu bar).
- **Test harness** — per-leaf temp store path; optional httptest; optional
  in-process agentcli under a mutex.

**Behaviors**

- Missing or empty file → one fixed root folder `id=root`, `name=Bookmarks`,
  empty `children`, `version=1`.
- Root folder is fixed: cannot delete `root`; default parent for add is `root`.
- IDs: server generates when omitted; client may supply stable id.
- Delete folder is recursive; delete url removes one node.
- Move reparents under `parent_id` with optional `index`.
- Validation: non-empty name; url type requires absolute http(s) (or reasonable
  absolute URL); browser ∈ {empty/null, default, chrome, firefox, opera}.
- Browser resolve: non-empty bookmark.browser wins; else globalDefault; if both
  empty → `"default"`.
- CLI empty list: human message includes `No bookmarks` (or shows empty root
  only); `--json` dumps document shape.
- `bookmarks open <id>`: resolve effective browser and build open plan without
  launching a GUI browser in CI (dry-run / captured argv). Prefer printing the
  effective browser name and URL on stdout when dry-run is active.
- Menu empty label: `No bookmarks`.

## Version

0.0.2

## Decision Tree

```
[bookmarks management — store + resolve + API + CLI + labels]
 |
 +-- store/                              (GROUP)  persistence + tree ops
 |    +-- load/
 |    |    +-- missing-file-default-root/  (LEAF)
 |    |    +-- round-trip-persist/         (LEAF)
 |    +-- add/
 |    |    +-- url-under-root/             (LEAF)
 |    |    +-- folder-under-root/          (LEAF)
 |    |    +-- custom-id/                  (LEAF)
 |    +-- update/
 |    |    +-- name-and-url/               (LEAF)
 |    |    +-- set-and-clear-browser/      (LEAF)
 |    +-- delete/
 |    |    +-- url-node/                   (LEAF)
 |    |    +-- folder-recursive/           (LEAF)
 |    |    +-- reject-root/                (LEAF)
 |    +-- move/
 |    |    +-- reparent-with-index/        (LEAF)
 |    +-- validation/
 |         +-- empty-name/                 (LEAF)
 |         +-- invalid-url/                (LEAF)
 |
 +-- resolve/                            (GROUP)  effective browser preference
 |    +-- bookmark-overrides-global/       (LEAF)
 |    +-- inherit-global/                  (LEAF)
 |    +-- empty-both-default/              (LEAF)
 |
 +-- api/                                (GROUP)  HTTP CRUD on Manager
 |    +-- get-empty-tree/                  (LEAF)
 |    +-- post-add-url/                    (LEAF)
 |    +-- patch-update-name/               (LEAF)
 |    +-- delete-non-root/                 (LEAF)
 |    +-- move-to-parent/                  (LEAF)
 |    +-- errors/
 |         +-- not-found/                  (LEAF)
 |         +-- bad-request-empty-name/     (LEAF)
 |
 +-- cli/                                (GROUP)  local-agent bookmarks …
 |    +-- help-top-level/                  (LEAF)
 |    +-- list-empty-human/                (LEAF)
 |    +-- list-json-tree/                  (LEAF)
 |    +-- add-url-and-list/                (LEAF)
 |    +-- add-folder/                      (LEAF)
 |    +-- set-rename/                      (LEAF)
 |    +-- delete-removes/                  (LEAF)
 |    +-- move-reparent/                   (LEAF)
 |    +-- open-resolves-browser/           (LEAF)
 |
 +-- label/                              (GROUP)  menu pure formatters
      +-- empty-menu/                      (LEAF)
      +-- url-title/                       (LEAF)
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `store/load/missing-file-default-root` | Missing file → version 1, single root `Bookmarks` empty children |
| 2 | `store/load/round-trip-persist` | Add url then reload from disk keeps id/name/url |
| 3 | `store/add/url-under-root` | Add url under default parent `root` |
| 4 | `store/add/folder-under-root` | Add folder under `root` with empty children |
| 5 | `store/add/custom-id` | Client-supplied id is preserved |
| 6 | `store/update/name-and-url` | Patch name + url on existing bookmark |
| 7 | `store/update/set-and-clear-browser` | Set browser then clear inherits again |
| 8 | `store/delete/url-node` | Delete url; sibling remains |
| 9 | `store/delete/folder-recursive` | Delete folder removes descendants |
| 10 | `store/delete/reject-root` | Delete `root` fails; root still present |
| 11 | `store/move/reparent-with-index` | Move url into folder at index 0 |
| 12 | `store/validation/empty-name` | Empty name rejected |
| 13 | `store/validation/invalid-url` | Relative/non-absolute url rejected |
| 14 | `resolve/bookmark-overrides-global` | bookmark `firefox` beats global `chrome` |
| 15 | `resolve/inherit-global` | empty bookmark → global `chrome` |
| 16 | `resolve/empty-both-default` | both empty → `default` |
| 17 | `api/get-empty-tree` | GET `/api/bookmarks` returns default document |
| 18 | `api/post-add-url` | POST add url; GET shows node under root |
| 19 | `api/patch-update-name` | PATCH renames node |
| 20 | `api/delete-non-root` | DELETE removes node |
| 21 | `api/move-to-parent` | POST move reparents |
| 22 | `api/errors/not-found` | PATCH unknown id → 404 |
| 23 | `api/errors/bad-request-empty-name` | POST empty name → 400 |
| 24 | `cli/help-top-level` | `bookmarks -h` documents subcommands |
| 25 | `cli/list-empty-human` | empty → human `No bookmarks` (or empty root only) |
| 26 | `cli/list-json-tree` | `--json` includes version + roots |
| 27 | `cli/add-url-and-list` | add then list shows name/url |
| 28 | `cli/add-folder` | add-folder creates folder node |
| 29 | `cli/set-rename` | set --name renames |
| 30 | `cli/delete-removes` | delete id then list omits it |
| 31 | `cli/move-reparent` | move under folder |
| 32 | `cli/open-resolves-browser` | open dry-run reports effective browser |
| 33 | `label/empty-menu` | FormatEmptyBookmarksLabel → `No bookmarks` |
| 34 | `label/url-title` | FormatBookmarkMenuTitle(name) → name |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Default empty document | store/load/missing-file-default-root, api/get-empty-tree, cli/list-empty-human |
| Persist round-trip | store/load/round-trip-persist |
| Add url / folder / custom id | store/add/*, cli/add-* |
| Update name/url/browser clear | store/update/*, api/patch-*, cli/set-* |
| Delete url / recursive / protect root | store/delete/* |
| Move + index | store/move/*, api/move-*, cli/move-* |
| Validation name/url | store/validation/*, api/errors/bad-request-* |
| Browser resolve | resolve/* , cli/open-resolves-browser |
| HTTP 404 | api/errors/not-found |
| CLI help | cli/help-top-level |
| CLI json | cli/list-json-tree |
| Menu labels | label/* |

## Expected package surface (implementer)

Package `github.com/xhd2015/ai-critic/server/bookmarks`:

- `NewManagerAt(path string) *Manager`
- `(*Manager) Load() (*Document, error)` — missing file → default root
- `(*Manager) Document() *Document` — current tree (after Load/mutations)
- `(*Manager) Add(parentID string, n *Node, index *int) (*Node, error)`
- `(*Manager) Update(id string, opts UpdateOpts) (*Node, error)`
- `(*Manager) Delete(id string) error` — reject `root`
- `(*Manager) Move(id, parentID string, index *int) error`
- `ResolveBrowser(bookmarkBrowser, globalDefault string) string`
- `FormatEmptyBookmarksLabel() string`
- `FormatBookmarkMenuTitle(name string) string`
- `RegisterAPIWithManager(mux *http.ServeMux, m *Manager)`
- Types: `Document{Version int; Roots []*Node}`,
  `Node{Type,ID,Name,URL string; Browser *string; Children []*Node}`,
  `UpdateOpts{Name,URL,Browser *string; ClearBrowser bool}`

CLI: `local-agent bookmarks` via agentcli LocalProfile. For `open`, support a
dry-run path (e.g. env `BOOKMARKS_OPEN_DRY_RUN=1` or flag) that prints effective
browser + URL without calling `/usr/bin/open`.

## How to Run

```sh
doctest vet ./tests/bookmarks
doctest test ./tests/bookmarks/...
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
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/ai-critic/server/bookmarks"
	"github.com/xhd2015/doctest/session"
)

// agentcliMu serializes in-process agentcli.Run + testhooks overrides.
var agentcliMu sync.Mutex

// Request selects one surface (Mode) and operation parameters.
// Modes: store | resolve | api | cli | label
type Request struct {
	Mode string

	// --- store ---
	// StoreOp: load | add | update | delete | move
	StoreOp      string
	SeedJSON     string // optional raw bookmarks.json before Load
	ParentID     string
	NodeType     string // folder | url
	ID           string
	Name         string
	URL          string
	Browser      string
	SetBrowser   bool
	ClearBrowser bool
	Index        *int
	MoveParentID string
	// SecondOp: "" | clear-browser | reload
	SecondOp string
	// PreAdds applied via Manager before StoreOp (same file)
	PreAdds []SeedNode

	// --- resolve ---
	BookmarkBrowser string
	GlobalDefault   string

	// --- api ---
	// APIOp: get | post | patch | delete | move
	APIOp    string
	RawBody  map[string]any
	QueryID  string
	SeedAdds []SeedNode

	// --- cli ---
	CLIArgs []string
	// ExtraEnv for child/in-process (e.g. BOOKMARKS_OPEN_DRY_RUN=1)
	CLIEnv []string

	// --- label ---
	// LabelKind: empty | url_title
	LabelKind string
	LabelName string

	Token string
}

type SeedNode struct {
	Type    string
	ID      string
	Name    string
	URL     string
	Browser string
	Parent  string // default root
}

type NodeView struct {
	Type     string     `json:"type"`
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	URL      string     `json:"url,omitempty"`
	Browser  *string    `json:"browser"`
	Children []NodeView `json:"children,omitempty"`
}

type DocView struct {
	Version int        `json:"version"`
	Roots   []NodeView `json:"roots"`
}

type Response struct {
	StorePath string
	ErrMsg    string

	Doc       *DocView
	Node      *NodeView
	RootCount int

	EffectiveBrowser string
	OpenDryRun       string

	HTTPStatus int
	Body       string
	BodyDoc    *DocView

	ExitCode int
	Stdout   string
	Stderr   string
	Combined string

	Label string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	if req.Token == "" {
		req.Token = "test-token"
	}
	if req.Mode == "" {
		return nil, fmt.Errorf("Request.Mode required")
	}
	switch req.Mode {
	case "store":
		return runStore(t, req)
	case "resolve":
		return runResolve(req)
	case "api":
		return runAPI(t, req)
	case "cli":
		return runCLI(t, req)
	case "label":
		return runLabel(req)
	default:
		return nil, fmt.Errorf("unknown Mode %q", req.Mode)
	}
}

func runStore(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	path := newStorePath(t)
	resp.StorePath = path
	if req.SeedJSON != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(req.SeedJSON), 0644); err != nil {
			return nil, err
		}
	}
	m := bookmarks.NewManagerAt(path)
	if _, err := m.Load(); err != nil {
		resp.ErrMsg = err.Error()
		return resp, nil
	}
	if err := applySeedAdds(m, req.PreAdds); err != nil {
		return nil, err
	}

	var opErr error
	switch req.StoreOp {
	case "load", "":
		// load only
	case "add":
		n := buildNode(req.NodeType, req.ID, req.Name, req.URL, req.SetBrowser, req.Browser)
		parent := req.ParentID
		if parent == "" {
			parent = "root"
		}
		added, err := m.Add(parent, n, req.Index)
		if err != nil {
			opErr = err
		} else {
			resp.Node = nodeToView(added)
		}
	case "update":
		opts := bookmarks.UpdateOpts{ClearBrowser: req.ClearBrowser}
		if req.Name != "" {
			n := req.Name
			opts.Name = &n
		}
		if req.URL != "" {
			u := req.URL
			opts.URL = &u
		}
		if req.SetBrowser {
			b := req.Browser
			opts.Browser = &b
		}
		updated, err := m.Update(req.ID, opts)
		if err != nil {
			opErr = err
		} else {
			resp.Node = nodeToView(updated)
		}
		if opErr == nil && req.SecondOp == "clear-browser" {
			cleared, err := m.Update(req.ID, bookmarks.UpdateOpts{ClearBrowser: true})
			if err != nil {
				opErr = err
			} else {
				resp.Node = nodeToView(cleared)
			}
		}
	case "delete":
		opErr = m.Delete(req.ID)
	case "move":
		parent := req.MoveParentID
		if parent == "" {
			parent = req.ParentID
		}
		opErr = m.Move(req.ID, parent, req.Index)
	default:
		return nil, fmt.Errorf("unknown StoreOp %q", req.StoreOp)
	}
	if opErr != nil {
		resp.ErrMsg = opErr.Error()
	}

	if req.SecondOp == "reload" {
		m2 := bookmarks.NewManagerAt(path)
		if _, err := m2.Load(); err != nil {
			resp.ErrMsg = err.Error()
			return resp, nil
		}
		resp.Doc = docToView(m2.Document())
	} else {
		resp.Doc = docToView(m.Document())
	}
	if resp.Doc != nil {
		resp.RootCount = len(resp.Doc.Roots)
	}
	return resp, nil
}

func runResolve(req *Request) (*Response, error) {
	return &Response{
		EffectiveBrowser: bookmarks.ResolveBrowser(req.BookmarkBrowser, req.GlobalDefault),
	}, nil
}

func runAPI(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	path := newStorePath(t)
	resp.StorePath = path
	m := bookmarks.NewManagerAt(path)
	if _, err := m.Load(); err != nil {
		return nil, err
	}
	if err := applySeedAdds(m, req.SeedAdds); err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	bookmarks.RegisterAPIWithManager(mux, m)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+req.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	base := srv.URL

	httpResp, err := performAPI(base, req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	b, _ := io.ReadAll(httpResp.Body)
	resp.HTTPStatus = httpResp.StatusCode
	resp.Body = string(b)
	if httpResp.StatusCode >= 400 {
		resp.ErrMsg = strings.TrimSpace(resp.Body)
	}

	getResp, err := doJSON(base, "GET", "/api/bookmarks", req.Token, nil)
	if err == nil {
		defer getResp.Body.Close()
		gb, _ := io.ReadAll(getResp.Body)
		if getResp.StatusCode == 200 {
			var dv DocView
			if json.Unmarshal(gb, &dv) == nil {
				resp.BodyDoc = &dv
				resp.Doc = &dv
			}
		}
	}
	if req.APIOp == "get" || req.APIOp == "" {
		var dv DocView
		if json.Unmarshal(b, &dv) == nil {
			resp.BodyDoc = &dv
			resp.Doc = &dv
		}
	}
	return resp, nil
}

func performAPI(base string, req *Request) (*http.Response, error) {
	switch req.APIOp {
	case "get", "":
		return doJSON(base, "GET", "/api/bookmarks", req.Token, nil)
	case "post":
		body := req.RawBody
		if body == nil {
			body = map[string]any{
				"parent_id": req.ParentID,
				"type":      req.NodeType,
				"name":      req.Name,
			}
			if req.ParentID == "" {
				body["parent_id"] = "root"
			}
			if req.URL != "" {
				body["url"] = req.URL
			}
			if req.SetBrowser {
				body["browser"] = req.Browser
			}
			if req.ID != "" {
				body["id"] = req.ID
			}
		}
		return doJSON(base, "POST", "/api/bookmarks", req.Token, body)
	case "patch":
		body := req.RawBody
		if body == nil {
			body = map[string]any{}
			if req.Name != "" {
				body["name"] = req.Name
			}
			if req.URL != "" {
				body["url"] = req.URL
			}
			if req.ClearBrowser {
				body["browser"] = ""
			} else if req.SetBrowser {
				body["browser"] = req.Browser
			}
		}
		q := req.QueryID
		if q == "" {
			q = req.ID
		}
		return doJSON(base, "PATCH", "/api/bookmarks?id="+q, req.Token, body)
	case "delete":
		q := req.QueryID
		if q == "" {
			q = req.ID
		}
		return doJSON(base, "DELETE", "/api/bookmarks?id="+q, req.Token, nil)
	case "move":
		body := req.RawBody
		if body == nil {
			body = map[string]any{
				"id":        req.ID,
				"parent_id": req.MoveParentID,
			}
			if req.Index != nil {
				body["index"] = *req.Index
			}
		}
		return doJSON(base, "POST", "/api/bookmarks/move", req.Token, body)
	default:
		return nil, fmt.Errorf("unknown APIOp %q", req.APIOp)
	}
}

func runCLI(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	home, err := os.MkdirTemp("", "bookmarks-cli-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	aiCritic := filepath.Join(home, ".ai-critic")
	if err := os.MkdirAll(aiCritic, 0755); err != nil {
		return nil, err
	}
	storePath := filepath.Join(aiCritic, "bookmarks.json")
	resp.StorePath = storePath

	if req.SeedJSON != "" {
		if err := os.WriteFile(storePath, []byte(req.SeedJSON), 0644); err != nil {
			return nil, err
		}
	}
	m := bookmarks.NewManagerAt(storePath)
	if _, err := m.Load(); err != nil {
		return nil, err
	}
	if err := applySeedAdds(m, req.SeedAdds); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	bookmarks.RegisterAPIWithManager(mux, m)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})
	mux.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authenticated":true,"initialized":true,"status":"ok"}`))
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" || r.URL.Path == "/api/auth/status" {
			mux.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+req.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	argv := append([]string{"--server", srv.URL, "--token", req.Token}, req.CLIArgs...)

	agentcliMu.Lock()
	defer agentcliMu.Unlock()

	testhooks.SetHomeOverride(home)
	testhooks.SetReachabilityForTest("up")
	defer testhooks.ResetInProcessOverrides()

	// Optional dry-run env for open (process env for in-process CLI reads)
	var envRestore []func()
	for _, kv := range req.CLIEnv {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := parts[0], parts[1]
		old, had := os.LookupEnv(k)
		_ = os.Setenv(k, v)
		envRestore = append(envRestore, func() {
			if had {
				_ = os.Setenv(k, old)
			} else {
				_ = os.Unsetenv(k)
			}
		})
	}
	defer func() {
		for i := len(envRestore) - 1; i >= 0; i-- {
			envRestore[i]()
		}
	}()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return nil, err
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutW, stderrW

	runErr := agentcli.Run(agentcli.LocalProfile(), argv)

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	outBytes, _ := io.ReadAll(stdoutR)
	errBytes, _ := io.ReadAll(stderrR)
	_ = stdoutR.Close()
	_ = stderrR.Close()

	resp.Stdout = string(outBytes)
	resp.Stderr = string(errBytes)
	if runErr != nil {
		resp.ExitCode = 1
		resp.ErrMsg = runErr.Error()
		resp.Stderr += fmt.Sprintf("Error: %v\n", runErr)
	}
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	// Re-load tree from the same manager path (CLI should have mutated via API or local file)
	m2 := bookmarks.NewManagerAt(storePath)
	if _, err := m2.Load(); err == nil {
		resp.Doc = docToView(m2.Document())
	}
	if len(req.CLIArgs) >= 2 && req.CLIArgs[0] == "bookmarks" && req.CLIArgs[1] == "open" {
		resp.OpenDryRun = resp.Combined
	}
	return resp, nil
}

func runLabel(req *Request) (*Response, error) {
	resp := &Response{}
	switch req.LabelKind {
	case "empty", "":
		resp.Label = bookmarks.FormatEmptyBookmarksLabel()
	case "url_title":
		resp.Label = bookmarks.FormatBookmarkMenuTitle(req.LabelName)
	default:
		return nil, fmt.Errorf("unknown LabelKind %q", req.LabelKind)
	}
	return resp, nil
}

func newStorePath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "bookmarks-store-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "bookmarks.json")
}

func buildNode(typ, id, name, url string, setBrowser bool, browser string) *bookmarks.Node {
	n := &bookmarks.Node{Type: typ, ID: id, Name: name, URL: url}
	if setBrowser {
		b := browser
		n.Browser = &b
	}
	return n
}

func applySeedAdds(m *bookmarks.Manager, seeds []SeedNode) error {
	for _, s := range seeds {
		parent := s.Parent
		if parent == "" {
			parent = "root"
		}
		n := &bookmarks.Node{Type: s.Type, ID: s.ID, Name: s.Name, URL: s.URL}
		if s.Browser != "" {
			b := s.Browser
			n.Browser = &b
		}
		if _, err := m.Add(parent, n, nil); err != nil {
			return fmt.Errorf("seed add %q: %w", s.Name, err)
		}
	}
	return nil
}

func doJSON(base, method, path, token string, body any) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, base+path, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return http.DefaultClient.Do(req)
}

func docToView(d *bookmarks.Document) *DocView {
	if d == nil {
		return nil
	}
	v := &DocView{Version: d.Version}
	for _, r := range d.Roots {
		v.Roots = append(v.Roots, *nodeToView(r))
	}
	return v
}

func nodeToView(n *bookmarks.Node) *NodeView {
	if n == nil {
		return nil
	}
	v := &NodeView{Type: n.Type, ID: n.ID, Name: n.Name, URL: n.URL, Browser: n.Browser}
	for _, c := range n.Children {
		v.Children = append(v.Children, *nodeToView(c))
	}
	return v
}

// FindNode walks DocView for id.
func FindNode(doc *DocView, id string) *NodeView {
	if doc == nil {
		return nil
	}
	var walk func(nodes []NodeView) *NodeView
	walk = func(nodes []NodeView) *NodeView {
		for i := range nodes {
			if nodes[i].ID == id {
				return &nodes[i]
			}
			if n := walk(nodes[i].Children); n != nil {
				return n
			}
		}
		return nil
	}
	return walk(doc.Roots)
}

// FindNodeByName walks DocView for first name match.
func FindNodeByName(doc *DocView, name string) *NodeView {
	if doc == nil {
		return nil
	}
	var walk func(nodes []NodeView) *NodeView
	walk = func(nodes []NodeView) *NodeView {
		for i := range nodes {
			if nodes[i].Name == name {
				return &nodes[i]
			}
			if n := walk(nodes[i].Children); n != nil {
				return n
			}
		}
		return nil
	}
	return walk(doc.Roots)
}

func RootChildren(doc *DocView) []NodeView {
	if doc == nil || len(doc.Roots) == 0 {
		return nil
	}
	return doc.Roots[0].Children
}

func defaultRootOK(doc *DocView) bool {
	if doc == nil || doc.Version != 1 || len(doc.Roots) != 1 {
		return false
	}
	r := doc.Roots[0]
	return r.ID == "root" && r.Type == "folder" && r.Name == "Bookmarks"
}
```
