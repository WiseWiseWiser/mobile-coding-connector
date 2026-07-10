# Local iTerm2 Open API Doctests

Server-side `POST /api/local/iterm2/open` that opens a directory in iTerm2 via
`github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2.OpenConfig` (same core as
`kool iterm2 -r <dir>`). No `kool` binary exec. Handler injects `OpenConfig` /
`Config.Osascript` for CI (no live iTerm).

Swift Projects/Terminals clients that call this API are covered by extensions
to `tests/macos-menubar-projects` and `tests/macos-menubar-terminals`.

# DSN (Domain Specific Notion)

**Participants**

- **HTTP handler (`server/localiterm2`)** — serves `POST /api/local/iterm2/open`;
  parses JSON `{dir, mode, send}`, maps mode strings to `iterm2.OpenMode`,
  validates `dir`, calls injectible `Open` (default `iterm2.OpenConfig`).
- **ParseOpenMode** — pure helper: `"reuse"`/`""` → `ModeReuseCurrent`,
  `"new"` → `ModeForceNew`, `"smart"` → `ModeSmart`; unknown → error.
- **shell/iterm2 OpenConfig** — builds AppleScript for reuse/new/smart and runs
  osascript; accepts `Config{Mode, FollowUpCommands, Osascript, Installed}`.
- **Auth middleware** — Bearer (or cookie) required for `/api/*` unless path is
  in the skip list; this endpoint is **not** skip-listed.
- **Host mux (`server.Serve`)** — registers the route beside other APIs.
- **Test harness** — pure parse calls; `httptest` handler/register with injected
  Open that records mode/follow-ups/script; auth via temp credentials +
  `auth.Middleware`; skip-list source inspection of `server/server.go`.

**Behaviors**

- Request: `dir` required; `mode` optional default `"reuse"`; `send` optional
  string array → `FollowUpCommands`.
- Mode map: omit/empty/`reuse` → `ModeReuseCurrent`; `new` → `ModeForceNew`;
  `smart` → `ModeSmart`. Note: lib zero-value `Config.Mode` is `ModeSmart` —
  handler must set reuse explicitly when defaulting.
- Success → `200` with **`{"ok":true}`** (locked response body).
- Missing/empty `dir` → **4xx** `{"error":"..."}` (client/validation).
- Path missing or not a directory → **4xx** `{"error":"..."}` (validate before Open).
- Injected Open/Osascript success → 200 + `ok:true`; recorded mode/script reflect
  path and mode semantics (reuse markers vs force-new vs smart).
- Injected Open error → **5xx** `{"error":"..."}` only after validation passed.
- No Bearer (and no cookie) under auth middleware → 401.
- Valid Bearer → handler runs (200 + `ok:true` when Open succeeds).
- `Register` mounts `POST /api/local/iterm2/open` (not 404).
- Path absent from auth skip list in `server.Serve`.

## Version

0.0.2

## Decision Tree

```
[local iterm2 open]
 |
 +-- parse-mode/                         (GROUP)  pure ParseOpenMode
 |    +-- omit-or-empty/                 (LEAF)   "" → ModeReuseCurrent
 |    +-- reuse/                         (LEAF)   "reuse" → ModeReuseCurrent
 |    +-- new/                           (LEAF)   "new" → ModeForceNew
 |    +-- smart/                         (LEAF)   "smart" → ModeSmart
 |    +-- invalid/                       (LEAF)   unknown → error
 |
 +-- open/                               (GROUP)  POST handler (inject Open)
 |    +-- success/                       (GROUP)  valid dir → 200
 |    |    +-- default-mode-reuse/       (LEAF)   omit mode → reuse
 |    |    +-- mode-reuse/               (LEAF)   mode=reuse
 |    |    +-- mode-new/                 (LEAF)   mode=new
 |    |    +-- mode-smart/               (LEAF)   mode=smart
 |    |    +-- with-send/                (LEAF)   send[] → FollowUpCommands
 |    +-- validation/                    (GROUP)  bad request → 4xx
 |    |    +-- missing-dir/              (LEAF)   no dir field
 |    |    +-- empty-dir/                (LEAF)   dir=""
 |    |    +-- not-directory/            (LEAF)   dir is a file
 |    |    +-- path-missing/             (LEAF)   dir does not exist
 |    +-- open-error/                    (GROUP)  Open fails → 5xx
 |         +-- inject-fail/              (LEAF)   injected Open error
 |
 +-- register/                           (GROUP)  mux mount + skip-list
 |    +-- route-mounted/                 (LEAF)   POST path not 404
 |    +-- not-in-auth-skip-list/         (LEAF)   server.go skip list omits path
 |
 +-- auth/                               (GROUP)  Bearer required
      +-- no-bearer/                     (LEAF)   no Authorization → 401
      +-- with-bearer/                   (LEAF)   valid Bearer → 200
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `parse-mode/omit-or-empty` | Empty/omit mode string → `ModeReuseCurrent` |
| 2 | `parse-mode/reuse` | `"reuse"` → `ModeReuseCurrent` |
| 3 | `parse-mode/new` | `"new"` → `ModeForceNew` |
| 4 | `parse-mode/smart` | `"smart"` → `ModeSmart` |
| 5 | `parse-mode/invalid` | Unknown mode → error |
| 6 | `open/success/default-mode-reuse` | Omit mode → reuse; 200 `{"ok":true}` |
| 7 | `open/success/mode-reuse` | mode=reuse; 200 `{"ok":true}` |
| 8 | `open/success/mode-new` | mode=new → ModeForceNew; ok:true |
| 9 | `open/success/mode-smart` | mode=smart → ModeSmart; ok:true |
| 10 | `open/success/with-send` | send → FollowUpCommands; ok:true |
| 11 | `open/validation/missing-dir` | No dir → 4xx + error |
| 12 | `open/validation/empty-dir` | Empty dir → 4xx + error |
| 13 | `open/validation/not-directory` | File path → **4xx** + error |
| 14 | `open/validation/path-missing` | Missing path → **4xx** + error |
| 15 | `open/open-error/inject-fail` | Open error → 5xx + error |
| 16 | `register/route-mounted` | Register → 200 ok (not 404) |
| 17 | `register/not-in-auth-skip-list` | Registered + not in auth skip list |
| 18 | `auth/no-bearer` | Middleware 401 without token |
| 19 | `auth/with-bearer` | Valid Bearer → 200 `{"ok":true}` |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| omit-or-empty | parse | ModeInput="" | Mode=ReuseCurrent, no err |
| reuse | parse | ModeInput=reuse | Mode=ReuseCurrent |
| new | parse | ModeInput=new | Mode=ForceNew |
| smart | parse | ModeInput=smart | Mode=Smart |
| invalid | parse | ModeInput=bogus | ParseErr |
| default-mode-reuse | open | dir=tmp, omit mode | 200 `{"ok":true}`, ModeReuseCurrent |
| mode-reuse | open | mode=reuse | 200 `{"ok":true}`, ModeReuseCurrent |
| mode-new | open | mode=new | 200 `{"ok":true}`, ModeForceNew |
| mode-smart | open | mode=smart | 200 `{"ok":true}`, ModeSmart |
| with-send | open | send=["echo hi"] | 200 ok + FollowUps |
| missing-dir | open | omit dir | 4xx + error |
| empty-dir | open | dir="" | 4xx + error |
| not-directory | open | dir=file | **4xx** + error |
| path-missing | open | dir=missing | **4xx** + error |
| inject-fail | open | Open returns err | 5xx + error |
| route-mounted | register | Register | 200 ok (not 404) |
| not-in-auth-skip-list | skip_list | server.go | registered + not skipped |
| no-bearer | auth | no header | 401 |
| with-bearer | auth | Bearer token | 200 ok |

## How to Run

```sh
doctest vet ./tests/local-iterm2-open
doctest test ./tests/local-iterm2-open/...
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
	"testing"

	"github.com/xhd2015/ai-critic/server/auth"
	"github.com/xhd2015/ai-critic/server/localiterm2"
	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
)

// openEndpoint is the fixed product path for local iTerm2 open.
const openEndpoint = "/api/local/iterm2/open"

// Request drives one localiterm2 scenario.
type Request struct {
	// Op: "parse" | "open" | "register" | "auth" | "skip_list"
	Op string

	// parse
	ModeInput string

	// open / auth body
	Dir           string
	Mode          string // JSON mode; empty + OmitMode → omit field
	OmitMode      bool
	OmitDir       bool
	Send          []string
	OmitSend      bool
	// InjectOpenError when non-empty makes the injected Open return that error.
	InjectOpenError string
	// UseRealOpenConfig runs iterm2.OpenConfig with injected Osascript/Installed
	// (records script). When false, injected Open only records Mode/FollowUps.
	UseRealOpenConfig bool

	// auth
	BearerToken     string
	OmitAuth        bool
	CredentialsPath string // set by Setup for auth leaves
}

// Response captures parse results, HTTP outcome, and injection observations.
type Response struct {
	// parse
	ParsedMode iterm2.OpenMode
	ParseErr   string

	// HTTP
	StatusCode  int
	Body        string
	ContentType string
	Error       string
	OK          bool

	// injection observations
	OpenCalled     bool
	RecordedDir    string
	RecordedMode   iterm2.OpenMode
	RecordedSend   []string
	RecordedScript string

	// skip_list / host registration contract
	InAuthSkipList      bool
	RegisteredInServer  bool // server.go references open endpoint / localiterm2
	SkipListSource      string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "parse":
		mode, err := localiterm2.ParseOpenMode(req.ModeInput)
		resp.ParsedMode = mode
		if err != nil {
			resp.ParseErr = err.Error()
		}
		return resp, nil
	case "open":
		return runOpen(t, req, resp, false)
	case "register":
		return runRegister(t, req, resp)
	case "auth":
		return runAuth(t, req, resp)
	case "skip_list":
		return runSkipList(t, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

type openCapture struct {
	called bool
	dir    string
	mode   iterm2.OpenMode
	send   []string
	script string
}

// openCaptureOpen returns an injectible Open that records into c.
// Free function (not a method): doctest nests helpers inside Test and drops receivers.
func openCaptureOpen(c *openCapture, req *Request) func(dir string, cfg *iterm2.Config) error {
	return func(dir string, cfg *iterm2.Config) error {
		c.called = true
		c.dir = dir
		if cfg != nil {
			c.mode = cfg.Mode
			c.send = append([]string(nil), cfg.FollowUpCommands...)
		}
		if req.InjectOpenError != "" {
			return fmt.Errorf("%s", req.InjectOpenError)
		}
		if !req.UseRealOpenConfig {
			return nil
		}
		// Call real lib with test hooks — no live iTerm.
		iterm2.SetGOOSForTest("darwin")
		defer iterm2.SetGOOSForTest("")
		libCfg := &iterm2.Config{
			Mode:             c.mode,
			FollowUpCommands: c.send,
			Installed:        func() bool { return true },
			Osascript: func(script string) error {
				c.script = script
				return nil
			},
		}
		return iterm2.OpenConfig(dir, libCfg)
	}
}

func applyCapture(resp *Response, c *openCapture) {
	resp.OpenCalled = c.called
	resp.RecordedDir = c.dir
	resp.RecordedMode = c.mode
	resp.RecordedSend = c.send
	resp.RecordedScript = c.script
}

func buildOpenBody(req *Request) ([]byte, error) {
	m := map[string]any{}
	if !req.OmitDir {
		m["dir"] = req.Dir
	}
	if !req.OmitMode {
		// Always include mode key when not omitted (may be empty string).
		m["mode"] = req.Mode
	}
	if !req.OmitSend && req.Send != nil {
		m["send"] = req.Send
	}
	return json.Marshal(m)
}

func runOpen(t *testing.T, req *Request, resp *Response, withAuth bool) (*Response, error) {
	cap := &openCapture{}
	h := &localiterm2.Handler{Open: openCaptureOpen(cap, req)}
	body, err := buildOpenBody(req)
	if err != nil {
		return nil, err
	}
	httpReq := httptest.NewRequest(http.MethodPost, openEndpoint, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if withAuth && !req.OmitAuth && req.BearerToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.BearerToken)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httpReq)
	fillHTTP(resp, rr)
	applyCapture(resp, cap)
	return resp, nil
}

func runRegister(t *testing.T, req *Request, resp *Response) (*Response, error) {
	cap := &openCapture{}
	// Ensure valid dir for register smoke so success path is reachable.
	if req.Dir == "" {
		req.Dir = t.TempDir()
	}
	req.OmitMode = true
	h := &localiterm2.Handler{Open: openCaptureOpen(cap, req)}
	mux := http.NewServeMux()
	localiterm2.Register(mux, h)
	body, err := buildOpenBody(req)
	if err != nil {
		return nil, err
	}
	httpReq := httptest.NewRequest(http.MethodPost, openEndpoint, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httpReq)
	fillHTTP(resp, rr)
	applyCapture(resp, cap)
	return resp, nil
}

func runAuth(t *testing.T, req *Request, resp *Response) (*Response, error) {
	if req.CredentialsPath == "" {
		return nil, fmt.Errorf("CredentialsPath required for auth op")
	}
	// Isolate credentials file for this leaf.
	prev := "" // auth has no getter; restore via empty rewrite is unsafe — set path only.
	auth.SetCredentialsFile(req.CredentialsPath)
	t.Cleanup(func() {
		// Best-effort: point back at default path constant if package allows.
		// Leaves use unique temp files; subsequent tests re-set as needed.
		_ = prev
	})

	cap := &openCapture{}
	if req.Dir == "" {
		req.Dir = t.TempDir()
	}
	req.OmitMode = true
	h := &localiterm2.Handler{Open: openCaptureOpen(cap, req)}
	mux := http.NewServeMux()
	localiterm2.Register(mux, h)

	// Same skip list as production except we only care that openEndpoint is NOT skipped.
	// Empty skip list forces auth on all /api/* paths.
	handler := auth.Middleware(mux, nil)

	body, err := buildOpenBody(req)
	if err != nil {
		return nil, err
	}
	httpReq := httptest.NewRequest(http.MethodPost, openEndpoint, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if !req.OmitAuth && req.BearerToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.BearerToken)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httpReq)
	fillHTTP(resp, rr)
	applyCapture(resp, cap)
	return resp, nil
}

func runSkipList(t *testing.T, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}
	srcPath := filepath.Join(moduleRoot, "server", "server.go")
	b, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("read server.go: %w", err)
	}
	src := string(b)
	resp.SkipListSource = srcPath
	// Detect whether openEndpoint appears inside the Middleware skip slice.
	// Heuristic: between auth.Middleware( and the closing of that call's slice,
	// look for the open path string.
	resp.InAuthSkipList = skipListContainsOpenPath(src)
	// Host must register the route (HandleFunc / localiterm2.Register / path literal outside skip list).
	resp.RegisteredInServer = strings.Contains(src, openEndpoint) ||
		strings.Contains(src, "localiterm2.Register") ||
		strings.Contains(src, "localiterm2.Handler") ||
		strings.Contains(src, "/api/local/iterm2/")
	return resp, nil
}

func skipListContainsOpenPath(serverSrc string) bool {
	// Find auth.Middleware( ... []string{ ... } ) region and check for open path.
	idx := strings.Index(serverSrc, "auth.Middleware")
	if idx < 0 {
		// If middleware wiring is absent, treat as not skip-listed (still need route).
		return strings.Contains(serverSrc, openEndpoint) && strings.Contains(serverSrc, "skip")
	}
	// Scan a window after Middleware for the skip slice.
	window := serverSrc[idx:]
	if len(window) > 2500 {
		window = window[:2500]
	}
	// Path is skip-listed only if the exact endpoint string appears in this window
	// as a string literal near skip paths.
	return strings.Contains(window, `"`+openEndpoint+`"`) ||
		strings.Contains(window, `"/api/local/iterm2/open"`)
}

func fillHTTP(resp *Response, rr *httptest.ResponseRecorder) {
	resp.StatusCode = rr.Code
	resp.Body = rr.Body.String()
	resp.ContentType = rr.Header().Get("Content-Type")
	var m map[string]any
	if json.Unmarshal(rr.Body.Bytes(), &m) == nil {
		if e, ok := m["error"].(string); ok {
			resp.Error = e
		}
		if ok, exists := m["ok"].(bool); exists {
			resp.OK = ok
		}
	}
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

// Silence unused import if body helpers expand.
var _ = io.EOF
```
