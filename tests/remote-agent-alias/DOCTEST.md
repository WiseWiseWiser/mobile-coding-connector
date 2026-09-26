# Remote-Agent Alias Doctests

Doctests for `remote-agent alias list|add|update|delete|install|which`, the
global `--alias NAME` flag, and `remote-agent config set`.

Most leaves are **L2 in-process** (`agentcli.RunWithWriters` with an injected
HOME and two local fake servers). One **L3 e2e** smoke (`e2e/wrapper-runs-real-binary`)
builds the product `remote-agent` binary, installs a generated wrapper, and runs
that wrapper through `sh` so the wrapper → binary → server path is covered for real.

# DSN (Domain Specific Notion)

**L2 in-process**: the harness gives every leaf a temp `HOME`
(`testhooks.SetHomeOverride`, no `os.Setenv`), two `httptest` servers
(`default` and `xdev`) that record `METHOD /path` plus the bearer token, and a
fake `remote-agent` first on `PATH` so wrapper exec resolution is deterministic
(the generated wrapper execs the bare name `remote-agent`).

**Participants**

- **CLI (`agentcli.RunWithWriters`)** — `remote-agent …` against the temp HOME.
- **default server** — the seeded default domain (`--token tok-default`).
- **xdev server** — the server the `xdev` alias targets (`--token tok-xdev`).
- **alias store** — `~/.ai-critic/remote-agent-aliases.json` (`{name, binary, server}`).
- **wrapper** — `~/.local/bin/xdev-agent`, a generated `#!/bin/sh` forwarder.
- **client config** — `~/.ai-critic/remote-agent-config.json` (domains + default).
- **Prelude** — `Request.Prelude` seeds default domain + `xdev` alias + its token,
  so resolve/update/delete leaves start from a realistic installed state.

**Behaviors**

- `alias add <name> --server URL` records the alias and installs the wrapper;
  it never talks to a server, and warns when the server has no saved token.
- The wrapper body is `#!/bin/sh` + generated marker + `exec remote-agent --alias <name> "$@"`,
  so the server lives in the alias store, not in the script.
- A file at the wrapper path that lacks the generated marker is never
  overwritten or removed without `--force`.
- `--alias NAME` targets that alias's server and uses the saved domain token;
  a plain invocation still uses the default domain.
- Tokens belong to servers: `config set` writes the server this invocation
  points at (subcommand `--server`/`--alias`, else the global `--alias`/`--server`),
  never echoes the token, and is local-only.
- `alias update --server` leaves the wrapper byte-identical; `--binary` renames
  the wrapper and removes the old generated file.
- `alias delete` removes the record, and the wrapper only when generated.
- Unknown aliases, conflicting `config set` targets, and bad names fail with
  `Error:` on stderr and exit 1.

## Version

0.0.1

## Decision Tree

```
[remote-agent alias]
 |
 +-- add/                                (GROUP) record + install
 |    +-- installs-wrapper/              (LEAF) store + wrapper + no-token warning
 |    +-- dry-run-writes-nothing/        (LEAF) plan only, no files
 |    +-- refuses-foreign-wrapper/       (LEAF) user script kept, exit 1
 |
 +-- resolve/                            (GROUP) --alias targeting
 |    +-- alias-targets-alias-server/    (LEAF) --alias xdev ping -> xdev + its token
 |    +-- plain-uses-default/            (LEAF) ping -> default (MECE partner)
 |
 +-- token/                              (GROUP) credentials
 |    +-- config-set-via-alias/          (LEAF) token written for the alias's server
 |
 +-- update/                             (GROUP) change an alias
 |    +-- server-keeps-wrapper-stable/   (LEAF) bytes unchanged, store updated
 |
 +-- delete/                             (GROUP) remove an alias
 |    +-- keeps-foreign-wrapper/         (LEAF) record gone, user script kept
 |
 +-- e2e/                                (GROUP) product binaries
      +-- wrapper-runs-real-binary/      (LEAF) L3: sh wrapper -> real binary -> xdev
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `add/installs-wrapper` | Store + generated wrapper; warns about the missing token |
| 2 | `add/dry-run-writes-nothing` | `--dry-run` plans the write but creates no files |
| 3 | `add/refuses-foreign-wrapper` | Existing user script is kept; add fails without `--force` |
| 4 | `resolve/alias-targets-alias-server` | `--alias xdev ping` hits xdev with the xdev token |
| 5 | `resolve/plain-uses-default` | Plain `ping` still hits the default domain |
| 6 | `token/config-set-via-alias` | `--alias xdev config set --token …` writes xdev's domain |
| 7 | `update/server-keeps-wrapper-stable` | Retarget updates the store, not the wrapper bytes |
| 8 | `delete/keeps-foreign-wrapper` | Delete removes the record but never a user script |
| 9 | `e2e/wrapper-runs-real-binary` | L3: generated wrapper execs the real binary |

## Runner

```sh
doctest vet ./tests/remote-agent-alias
doctest test ./tests/remote-agent-alias/...
doctest test --label e2e ./tests/remote-agent-alias/...
```

## Harness

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/doctest/session"
)

const (
	aliasLeafName     = "xdev"
	aliasLeafBinary   = "xdev-agent"
	aliasDefaultToken = "tok-default"
	aliasLeafToken    = "tok-xdev"
	aliasForeignBody  = "#!/bin/sh\necho mine\n"
	aliasMarkerPrefix = "# generated by remote-agent alias install (alias: "

	// Leaves cannot know the httptest URLs or the leaf HOME (created inside
	// Run), so Args and WrapperArgs may use these placeholders.
	aliasURLPlaceholder   = "{{alias-url}}"
	defaultURLPlaceholder = "{{default-url}}"
	aliasBinPlaceholder   = "{{wrapper-bin}}"
)

// aliasHarnessMu serializes leaves: HOME resolution (testhooks) and the CLI's
// active profile are process-global, so parallel leaves would clobber each
// other's temp home. Hold it for the whole Run, and clear the override before
// releasing it (a t.Cleanup would race a concurrently running leaf).
var aliasHarnessMu sync.Mutex

// aliasFakeServer records every request it receives.
type aliasFakeServer struct {
	name string

	mu       sync.Mutex
	requests []string
	auth     []string

	srv *httptest.Server
}

func newAliasFakeServer(t *testing.T, name string) *aliasFakeServer {
	t.Helper()
	f := &aliasFakeServer{name: name}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		f.mu.Lock()
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		f.auth = append(f.auth, token)
		f.mu.Unlock()

		switch r.URL.Path {
		case "/ping":
			w.Write([]byte("pong"))
		case "/api/auth/status":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{\"initialized\":true,\"status\":\"ok\"}"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *aliasFakeServer) URL() string { return f.srv.URL }

func (f *aliasFakeServer) Hits() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *aliasFakeServer) Tokens() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.auth...)
}

// Request configures one leaf.
type Request struct {
	// Op selects what the harness runs: "" or "cli" → Args through the CLI;
	// "wrapper" → execute the installed wrapper with WrapperArgs.
	Op string

	Args    []string
	Prelude bool // seed default domain + xdev alias + xdev token before Args

	// SeedForeignWrapper pre-creates a user-owned script at the wrapper path.
	SeedForeignWrapper bool
	// MutateWrapper replaces the wrapper content after the Prelude (simulating
	// a user editing the generated file).
	MutateWrapper string

	// Op "wrapper"
	WrapperArgs []string
	// Stdin is injected into the wrapper child process (never process stdin).
	Stdin string
	// BuildBinary builds ./cmd/remote-agent and installs the alias with --bin,
	// so the wrapper execs a real product binary (L3).
	BuildBinary bool
}

// Response captures one leaf's observable state.
type Response struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Combined string

	Home       string
	Wrapper    string
	WrapperBin string
	ClientBin  string

	DefaultURL  string
	AliasURL    string
	DefaultHits int
	AliasHits   int
	DefaultAuth []string
	AliasAuth   []string

	WrapperBefore string
	WrapperAfter  string
	StoreJSON     string
	ConfigJSON    string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	if req.Op == "" {
		req.Op = "cli"
	}
	resp := &Response{}

	aliasHarnessMu.Lock()
	defer func() {
		testhooks.ResetInProcessOverrides()
		aliasHarnessMu.Unlock()
	}()

	home, err := os.MkdirTemp("", "remote-agent-alias-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	resp.Home = home
	testhooks.SetHomeOverride(home)

	// The wrapper execs an explicit --bin, so no PATH mutation is needed and
	// the generated exec line is deterministic.
	binDir := filepath.Join(home, "path-bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return nil, err
	}
	fakeAgent := filepath.Join(binDir, "remote-agent")
	if err := os.WriteFile(fakeAgent, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		return nil, err
	}
	resp.WrapperBin = fakeAgent
	resp.Wrapper = filepath.Join(home, ".local", "bin", aliasLeafBinary)

	def := newAliasFakeServer(t, "default")
	ali := newAliasFakeServer(t, "xdev")
	resp.DefaultURL, resp.AliasURL = def.URL(), ali.URL()

	if req.SeedForeignWrapper {
		if err := os.MkdirAll(filepath.Dir(resp.Wrapper), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(resp.Wrapper, []byte(aliasForeignBody), 0755); err != nil {
			return nil, err
		}
	}

	seed := func(args ...string) error {
		code, out, errOut := runAliasCLI(t, args...)
		if code != 0 {
			return fmt.Errorf("seed %v failed (exit %d)\n%s%s", args, code, out, errOut)
		}
		return nil
	}

	if req.Prelude {
		if err := seed("config", "set", "--server", def.URL(), "--token", aliasDefaultToken, "--default"); err != nil {
			return nil, err
		}
		bin := fakeAgent
		if req.BuildBinary {
			built, err := buildRemoteAgent(t, d, home)
			if err != nil {
				return nil, err
			}
			resp.ClientBin = built
			bin = built
		}
		if err := seed("alias", "add", aliasLeafName, "--server", ali.URL(), "--bin", bin); err != nil {
			return nil, err
		}
		if err := seed("config", "set", "--alias", aliasLeafName, "--token", aliasLeafToken); err != nil {
			return nil, err
		}
	}

	if req.MutateWrapper != "" {
		if err := os.WriteFile(resp.Wrapper, []byte(req.MutateWrapper), 0755); err != nil {
			return nil, err
		}
	}
	resp.WrapperBefore = readFileOrEmpty(resp.Wrapper)

	req.Args = expandAliasPlaceholders(req.Args, def.URL(), ali.URL(), resp.WrapperBin)
	req.WrapperArgs = expandAliasPlaceholders(req.WrapperArgs, def.URL(), ali.URL(), resp.WrapperBin)

	switch req.Op {
	case "cli":
		resp.ExitCode, resp.Stdout, resp.Stderr = runAliasCLI(t, req.Args...)
	case "wrapper":
		resp.ExitCode, resp.Stdout, resp.Stderr = runAliasWrapper(t, resp, req)
	default:
		return nil, fmt.Errorf("unknown Op %q", req.Op)
	}

	resp.Combined = resp.Stdout + resp.Stderr
	resp.WrapperAfter = readFileOrEmpty(resp.Wrapper)
	resp.StoreJSON = readFileOrEmpty(filepath.Join(home, ".ai-critic", "remote-agent-aliases.json"))
	resp.ConfigJSON = readFileOrEmpty(filepath.Join(home, ".ai-critic", "remote-agent-config.json"))
	resp.DefaultHits, resp.AliasHits = def.Hits(), ali.Hits()
	resp.DefaultAuth, resp.AliasAuth = def.Tokens(), ali.Tokens()
	return resp, nil
}

func readFileOrEmpty(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// expandAliasPlaceholders substitutes the leaf-unknown values in args.
func expandAliasPlaceholders(args []string, defaultURL, aliasURL, wrapperBin string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, 0, len(args))
	for _, a := range args {
		a = strings.ReplaceAll(a, defaultURLPlaceholder, defaultURL)
		a = strings.ReplaceAll(a, aliasURLPlaceholder, aliasURL)
		a = strings.ReplaceAll(a, aliasBinPlaceholder, wrapperBin)
		out = append(out, a)
	}
	return out
}

// runAliasCLI runs the CLI in-process with injected writers and mirrors the
// product binary's "Error: …" + non-zero exit convention.
func runAliasCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer

	runErr := agentcli.RunWithWriters(agentcli.RemoteProfile(), args, &stdout, &stderr)
	code := 0
	if runErr != nil {
		fmt.Fprintf(&stderr, "Error: %v\n", runErr)
		code = 1
	}
	return code, stdout.String(), stderr.String()
}

// runAliasWrapper executes the installed wrapper through sh, the way a shell
// would, with the leaf HOME and any leaf stdin.
func runAliasWrapper(t *testing.T, resp *Response, req *Request) (int, string, string) {
	t.Helper()
	cmd := exec.Command("sh", append([]string{resp.Wrapper}, req.WrapperArgs...)...)
	cmd.Env = append(os.Environ(), "HOME="+resp.Home)
	if req.Stdin != "" {
		cmd.Stdin = strings.NewReader(req.Stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Fprintf(&stderr, "Error: %v\n", err)
		}
	}
	return cmd.ProcessState.ExitCode(), stdout.String(), stderr.String()
}

// buildRemoteAgent builds the product CLI once per leaf into the leaf HOME.
func buildRemoteAgent(t *testing.T, d *session.Doctest, home string) (string, error) {
	t.Helper()
	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", err
	}
	bin := filepath.Join(binDir, "remote-agent")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/remote-agent")
	cmd.Dir = moduleRoot
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("build remote-agent: %v\n%s", err, buf.String())
	}
	return bin, nil
}

// aliasStoreNames returns the alias names recorded in the leaf's store.
func aliasStoreNames(t *testing.T, storeJSON string) []string {
	t.Helper()
	if strings.TrimSpace(storeJSON) == "" {
		return nil
	}
	var st struct {
		Aliases []struct {
			Name string `json:"name"`
		} `json:"aliases"`
	}
	if err := json.Unmarshal([]byte(storeJSON), &st); err != nil {
		t.Fatalf("parse alias store: %v\n%s", err, storeJSON)
	}
	names := make([]string, 0, len(st.Aliases))
	for _, a := range st.Aliases {
		names = append(names, a.Name)
	}
	return names
}

// aliasStoreServers maps alias name → server.
func aliasStoreServers(t *testing.T, storeJSON string) map[string]string {
	t.Helper()
	var st struct {
		Aliases []struct {
			Name   string `json:"name"`
			Server string `json:"server"`
		} `json:"aliases"`
	}
	if err := json.Unmarshal([]byte(storeJSON), &st); err != nil {
		t.Fatalf("parse alias store: %v\n%s", err, storeJSON)
	}
	out := map[string]string{}
	for _, a := range st.Aliases {
		out[a.Name] = a.Server
	}
	return out
}
```
