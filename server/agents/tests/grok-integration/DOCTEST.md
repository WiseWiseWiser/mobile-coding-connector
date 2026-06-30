# Grok Agent Integration Doctests

Package-level tests for `server/agents` verifying Grok as a first-class headless
agent: API listing, installed parity with OpenCode, session launch via opencode
serve, and Grok-specific model preference.

# DSN (Domain Specific Notion)

The grok-integration harness models ai-critic's Agent view backend: a catalog of
coding agents, headless session launch, and opencode-backed HTTP proxy.

**Participants**

- **Agent catalog** — `agentDefs` slice; `GET /api/agents` returns JSON with
  dynamic `installed` flags.
- **Session manager** — `sessionMgr.launch(agent_id, project_dir)` starts
  `opencode serve` on a free port or returns validation errors.
- **OpenCode child** — headless server exposing `/global/health`, `/config`,
  `/config/providers`; Grok sessions reuse the same binary path as OpenCode.
- **Model applier** — after ready, patches `/config` with saved model or a
  provider model matching an agent-specific substring (`grok` for agent `grok`).

**Behaviors**

- Catalog includes `id:grok`, `headless:true`, `command:opencode`.
- `installed` for `grok` matches `installed` for `opencode` (same binary resolution).
- Launch `grok` with valid project dir and opencode in PATH → session `running`.
- Launch `grok` without opencode → error mentioning install/binary.
- Unknown `agent_id` or non-directory `project_dir` → launch error (HTTP 400 path).
- When config model is empty and no saved model, grok sessions prefer a model ID
  containing `grok` (case-insensitive).

## Version

0.0.2

## Decision Tree

```
[grok agent integration]
 |
 +-- agent-list/                              (grouping)
 |    +-- includes-grok-entry/               (LEAF)  GET list has grok + fields
 |    +-- grok-installed-mirrors-opencode/   (grouping)
 |         +-- both-absent-not-installed/    (LEAF)  no opencode → both false
 |         +-- both-present-installed/       (LEAF)  fake opencode → both true
 |
 +-- session-launch/                          (grouping)
 |    +-- launch-grok-creates-session/        (LEAF)  mock serve → running session
 |    +-- launch-grok-without-opencode-fails/ (LEAF)  empty PATH → error
 |    +-- unknown-agent-id-rejected/          (LEAF)  agent_id=not-real
 |    +-- invalid-project-dir-rejected/       (LEAF)  project_dir=file not dir
 |
 +-- model-preference/                        (grouping)
      +-- grok-substring-not-kimi/            (LEAF)  preferred substring for grok
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `agent-list/includes-grok-entry` | List agents includes grok with headless and opencode command |
| 2 | `agent-list/grok-installed-mirrors-opencode/both-absent-not-installed` | Without opencode, grok and opencode installed=false |
| 3 | `agent-list/grok-installed-mirrors-opencode/both-present-installed` | With fake opencode on PATH, grok and opencode installed=true |
| 4 | `session-launch/launch-grok-creates-session` | launch("grok", dir) returns session with agent_id grok |
| 5 | `session-launch/launch-grok-without-opencode-fails` | launch fails when opencode not resolvable |
| 6 | `session-launch/unknown-agent-id-rejected` | launch unknown agent returns error |
| 7 | `session-launch/invalid-project-dir-rejected` | launch with file path returns error |
| 8 | `model-preference/grok-substring-not-kimi` | Grok agent uses grok model substring, not default kimi |

## How to Run

```sh
doctest vet ./server/agents/tests/grok-integration
doctest test ./server/agents/tests/grok-integration/...
go run ./script/build
```

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xhd2015/ai-critic/server/agents"
)

const (
	OpListAgents       = "list-agents"
	OpLaunchGrok       = "launch-grok"
	OpModelSubstring   = "model-substring"
)

type Request struct {
	Op string

	AgentID    string
	ProjectDir string

	UseFakeOpenCode bool
	StripOpenCode   bool

	UnknownAgentID bool
	InvalidProject bool
}

type AgentListEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Installed   bool   `json:"installed"`
	Headless    bool   `json:"headless"`
}

type Response struct {
	ListBody       string
	ListAgents     []AgentListEntry
	GrokDef        *AgentListEntry
	OpenCodeDef    *AgentListEntry

	LaunchSession *agents.AgentSessionInfo
	LaunchErr     error

	ModelSubstring string
	FakeBinDir     string
}

func prepStripPATH(t *testing.T) {
	agents.TestExported_StripOpencodeResolutionForDoctest(t)
	orig := os.Getenv("PATH")
	binDir := filepath.Join(t.TempDir(), "empty-bin")
	_ = os.MkdirAll(binDir, 0755)
	os.Setenv("PATH", binDir)
	t.Cleanup(func() { os.Setenv("PATH", orig) })
}

func installFakeOpenCodeServe(t *testing.T, resp *Response) error {
	root := DOCTEST_ROOT
	if root == "" {
		root = os.Getenv("DOCTEST_ROOT")
	}
	if root == "" {
		root = "."
	}
	srcDir := filepath.Join(root, "testdata", "fake-opencode")
	binDir := filepath.Join(t.TempDir(), "fake-opencode-bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	resp.FakeBinDir = binDir

	binPath := filepath.Join(binDir, "opencode")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = srcDir
	if out, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("build fake opencode: %v\n%s", err, out)
	}

	orig := os.Getenv("PATH")
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+orig)
	t.Cleanup(func() { os.Setenv("PATH", orig) })
	return nil
}

func runListAgents(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	if req.StripOpenCode {
		prepStripPATH(t)
	}
	if req.UseFakeOpenCode {
		if err := installFakeOpenCodeServe(t, resp); err != nil {
			return nil, err
		}
	}

	rr := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest(http.MethodGet, "/api/agents", nil)
	mux := http.NewServeMux()
	agents.RegisterAPI(mux)
	mux.ServeHTTP(rr, reqHTTP)

	if rr.Code != http.StatusOK {
		return nil, fmt.Errorf("list agents status %d: %s", rr.Code, rr.Body.String())
	}
	resp.ListBody = rr.Body.String()
	if err := json.Unmarshal(rr.Body.Bytes(), &resp.ListAgents); err != nil {
		return nil, err
	}
	for i := range resp.ListAgents {
		a := &resp.ListAgents[i]
		if a.ID == "grok" {
			resp.GrokDef = a
		}
		if a.ID == "opencode" {
			resp.OpenCodeDef = a
		}
	}
	return resp, nil
}

func runLaunchGrok(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	projectDir := req.ProjectDir
	if req.InvalidProject {
		f, err := os.CreateTemp("", "not-a-dir-*")
		if err != nil {
			return nil, err
		}
		f.Close()
		projectDir = f.Name()
		t.Cleanup(func() { os.Remove(projectDir) })
	} else if projectDir == "" {
		var err error
		projectDir, err = os.MkdirTemp("", "grok-proj-*")
		if err != nil {
			return nil, err
		}
		t.Cleanup(func() { os.RemoveAll(projectDir) })
	}

	agentID := "grok"
	if req.UnknownAgentID {
		agentID = "not-a-real-agent"
	}

	if req.StripOpenCode {
		prepStripPATH(t)
	}
	if req.UseFakeOpenCode {
		if err := installFakeOpenCodeServe(t, resp); err != nil {
			return nil, err
		}
	}

	info, err := agents.TestExported_LaunchAgentSession(agentID, projectDir, "")
	resp.LaunchErr = err
	if err == nil {
		resp.LaunchSession = &info
		t.Cleanup(func() { agents.TestExported_StopAgentSession(info.ID) })
	}
	return resp, nil
}

func runModelSubstring(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	resp.ModelSubstring = agents.TestExported_PreferredModelSubstringForAgent("grok")
	return resp, nil
}

func Run(t *testing.T, req *Request) (*Response, error) {
	if req.Op == "" {
		return nil, fmt.Errorf("Op is required")
	}

	switch req.Op {
	case OpListAgents:
		return runListAgents(t, req)
	case OpLaunchGrok:
		return runLaunchGrok(t, req)
	case OpModelSubstring:
		return runModelSubstring(t, req)
	default:
		return nil, fmt.Errorf("unknown Op: %s", req.Op)
	}
}
```