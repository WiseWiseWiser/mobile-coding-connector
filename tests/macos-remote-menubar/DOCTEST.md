# Remote macOS Menu Bar App Doctests

Pure-function and source-contract tests for the **remote** macOS menu-bar app
entry (`ai-critic-remote-macos.app` / **AI Critic(Remote)**), parallel to
`local-agent` vs `remote-agent`. Go helpers under `macosapp/remoteconfig` and
`macosapp/appprofile` are the doctest surface; Swift dual-product and
`install-remote.sh` are locked by read-only source-contract leaves.

# DSN (Domain Specific Notion)

**Participants**

- **Remote config package (`macosapp/remoteconfig`)** — pure helpers that load and
  save `~/.ai-critic/remote-agent-config.json` (same schema as `cmd/agentcli`
  `agentConfig`), resolve a default domain to a normalized endpoint + token,
  format `Authorization` headers, map connection state to guided status copy,
  and produce the Open-in-Browser base URL from a resolved remote endpoint.
- **App profile package (`macosapp/appprofile`)** — static `local` vs `remote`
  profile flags: whether the app spawns a keep-alive daemon, uses Bearer auth,
  which config file name / bundle id / display name apply.
- **Remote menu-bar app (Swift)** — product `ai-critic-remote-macos` that does
  **not** spawn a local daemon, does **not** show Restart Daemon, reads/writes
  the shared remote-agent config file, and calls remote server APIs with
  `Authorization: Bearer <token>`.
- **Install / bundle scripts** — `script/macos-app/install-remote.sh` (and remote
  bundle mode) produce `ai-critic-remote-macos.app` with bundle id
  `com.xhd2015.ai-critic-remote-macos` without embedding the `ai-critic` server
  binary.
- **Test harness** — invokes pure Go helpers with leaf JSON/temp-dir inputs, or
  inspects scripts/Swift sources; no network required for resolve/save/status/
  header/profile leaves.

**Behaviors**

- **Resolve**: missing file / unreadable / empty `domains` → state
  `not_configured`, no endpoint. Default matches a domain after trim + trailing-
  slash normalize → that server+token, state `ok` (config-level). Empty default
  with exactly one domain → that domain. Multiple domains with empty/missing
  default match → `no_default`. Normalized server has no trailing slash.
- **Save**: writes JSON mode `0600`; updating domains/default/token must preserve
  `project_bindings` (and not wipe unrelated list fields).
- **Auth header**: non-empty token → `Bearer <token>`; empty token → empty header
  value (no `Bearer` prefix).
- **Status copy**: guided strings for `not_configured`, `no_default`, `ok`,
  `unauthorized`, `unreachable`; `ok` is `Connected to {server}`; never include
  the raw token.
- **Profile**: remote → `SpawnsDaemon=false`, `UsesAuthToken=true`, config
  `remote-agent-config.json`, display name contains `Remote`; local →
  `SpawnsDaemon=true` (local app intent unchanged).
- **Browser URL**: uses resolved remote server base URL, not loopback keep-alive.
- **Client contracts**: remote menu must not expose Restart Daemon; install-remote
  identity matches product table and does not require embedding server binary.

## Version

0.0.2

## Decision Tree

```
[remote macOS menu bar]
 |
 +-- resolve/                          (GROUP)  config → endpoint + config state
 |    +-- missing-file/                (LEAF)   no file → not_configured
 |    +-- empty-domains/               (LEAF)   domains=[] → not_configured
 |    +-- default-matches/             (LEAF)   default matches domain
 |    +-- trailing-slash-match/        (LEAF)   slash-normalized match
 |    +-- whitespace-normalize/        (LEAF)   trim space + slash for match
 |    +-- first-domain-fallback/       (LEAF)   empty default, one domain
 |    +-- multi-domain-no-default/     (LEAF)   two domains, empty default → no_default
 |
 +-- save/                             (GROUP)  write remote-agent-config.json
 |    +-- preserves-project-bindings/  (LEAF)   update token keeps bindings
 |    +-- file-mode-0600/              (LEAF)   written file mode is 0600
 |
 +-- auth/                             (GROUP)  Authorization header helper
 |    +-- bearer-token/                (LEAF)   token abc → Bearer abc
 |    +-- empty-token/                 (LEAF)   empty token → empty header
 |
 +-- status/                           (GROUP)  guided connection status copy
 |    +-- not-configured/              (LEAF)   guide to Configure…
 |    +-- no-default/                  (LEAF)   multi-domain guidance
 |    +-- ok/                          (LEAF)   Connected to {server}, no token
 |    +-- unauthorized/                (LEAF)   token rejected + Configure…
 |    +-- unreachable/                 (LEAF)   cannot reach host + retry/test
 |
 +-- profile/                          (GROUP)  local vs remote app flags
 |    +-- remote/                      (LEAF)   no daemon, remote config + names
 |    +-- local/                       (LEAF)   SpawnsDaemon=true (unchanged)
 |
 +-- browser/                          (GROUP)  Open in Browser URL
 |    +-- resolved-remote-url/         (LEAF)   remote base, not localhost
 |
 +-- client/                           (GROUP)  Swift/script source contracts
      +-- no-restart-daemon/           (LEAF)   remote product hides Restart Daemon
      +-- install-remote-identity/     (LEAF)   install-remote app name + bundle id
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `resolve/missing-file` | Missing config file → `not_configured`, no endpoint |
| 2 | `resolve/empty-domains` | Empty `domains` → `not_configured` |
| 3 | `resolve/default-matches` | Default matches domain → server+token |
| 4 | `resolve/trailing-slash-match` | `https://x.com/` default matches `https://x.com` |
| 5 | `resolve/whitespace-normalize` | Spaced default matches trimmed domain |
| 6 | `resolve/first-domain-fallback` | Empty default, one domain → that domain |
| 7 | `resolve/multi-domain-no-default` | Two domains, empty default → `no_default` |
| 8 | `save/preserves-project-bindings` | Save after token update keeps `project_bindings` |
| 9 | `save/file-mode-0600` | Saved file has mode `0600` |
| 10 | `auth/bearer-token` | `AuthorizationHeader("abc")` → `Bearer abc` |
| 11 | `auth/empty-token` | Empty token → empty header string |
| 12 | `status/not-configured` | Status copy guides to Configure… |
| 13 | `status/no-default` | Status copy for multi-domain no default |
| 14 | `status/ok` | `Connected to {server}`; token not present |
| 15 | `status/unauthorized` | Token rejected + Configure… guidance |
| 16 | `status/unreachable` | Cannot reach host + retry/test guidance |
| 17 | `profile/remote` | Remote profile flags and display/bundle identity |
| 18 | `profile/local` | Local profile still spawns daemon |
| 19 | `browser/resolved-remote-url` | Browser URL is remote server, not loopback |
| 20 | `client/no-restart-daemon` | Remote product sources gate/hide Restart Daemon |
| 21 | `client/install-remote-identity` | install-remote.sh app name + bundle id + no server embed |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Config presence / empty domains | missing-file, empty-domains |
| Default match (exact / slash / whitespace) | default-matches, trailing-slash-match, whitespace-normalize |
| Single-domain fallback | first-domain-fallback |
| Multi-domain without default | multi-domain-no-default |
| Save preserves bindings | preserves-project-bindings |
| Save file permissions | file-mode-0600 |
| Auth header empty vs set | bearer-token, empty-token |
| Status state enum | not-configured, no-default, ok, unauthorized, unreachable |
| Token never in status | ok (and all status leaves) |
| Profile local vs remote | profile/local, profile/remote |
| Open browser base URL | resolved-remote-url |
| Remote menu / install identity | no-restart-daemon, install-remote-identity |

## How to Run

```sh
doctest vet ./tests/macos-remote-menubar
doctest test ./tests/macos-remote-menubar/...
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

	"github.com/xhd2015/ai-critic/macosapp/appprofile"
	"github.com/xhd2015/ai-critic/macosapp/remoteconfig"
)

// Stable status-copy contract (implementer must match exactly).
const (
	statusNotConfigured = "Not configured — open Configure… to add a remote server"
	statusNoDefault     = "Multiple servers configured — open Configure… to pick a default"
	statusUnauthorized  = "Token rejected — open Configure… to update credentials"
)

type Request struct {
	Op string

	// resolve / save: JSON body of remote-agent-config.json
	ConfigJSON string

	// load-resolve: load from ConfigPath (missing file ok)
	// save: write to ConfigPath under a temp dir when empty → Run creates path
	ConfigPath string
	UseLoad    bool // if true, load from ConfigPath before resolve

	// save mutation applied after load (or after parsing ConfigJSON)
	UpdateServer string
	UpdateToken  string
	UpdateDefault string

	// auth
	Token string

	// status
	ConnectionState string
	StatusServer    string // server shown in status (never a token)

	// profile: "local" | "remote"
	ProfileName string

	// browser: resolved server to open
	BrowserServer string
	BrowserToken  string
	BrowserOK     bool

	// client contract leaf id
	ClientLeaf string
}

type Response struct {
	// resolve
	State    string
	Resolved bool
	Server   string
	Token    string

	// auth
	AuthHeader string

	// status
	StatusLine           string
	StatusContainsToken  bool
	StatusContainsConfig bool // true if copy mentions Configure

	// profile
	SpawnsDaemon   bool
	UsesAuthToken  bool
	ConfigFileName string
	BundleID       string
	AppName        string
	DisplayName    string

	// browser
	BrowserURL string

	// save
	ProjectBindingsJSON string
	FileMode            os.FileMode
	SavedOK             bool

	// client
	HasRestartDaemonMenu bool
	RestartDaemonGated   bool // true if Restart Daemon only when SpawnsDaemon/local
	InstallAppName       string
	InstallBundleID      string
	EmbedsServerBinary   bool
	SourcesChecked       []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "resolve":
		return runResolve(t, req, resp)
	case "save":
		return runSave(t, req, resp)
	case "auth":
		resp.AuthHeader = remoteconfig.AuthorizationHeader(req.Token)
		return resp, nil
	case "status":
		line := remoteconfig.FormatStatus(remoteconfig.ConnectionState(req.ConnectionState), req.StatusServer)
		resp.StatusLine = line
		// Detect accidental token leakage if leaf provided a sentinel token via StatusServer misuse —
		// leaves pass server only; harness also checks common secret substrings when Token set.
		if req.Token != "" && strings.Contains(line, req.Token) {
			resp.StatusContainsToken = true
		}
		if strings.Contains(line, "Configure") {
			resp.StatusContainsConfig = true
		}
		return resp, nil
	case "profile":
		return runProfile(t, req, resp)
	case "browser":
		ep := remoteconfig.ResolvedEndpoint{
			Server: req.BrowserServer,
			Token:  req.BrowserToken,
			OK:     req.BrowserOK,
		}
		resp.BrowserURL = remoteconfig.OpenBrowserURL(ep)
		return resp, nil
	case "client":
		return runClientContract(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runResolve(t *testing.T, req *Request, resp *Response) (*Response, error) {
	var cfg *remoteconfig.Config
	if req.UseLoad {
		path := req.ConfigPath
		if path == "" {
			return nil, fmt.Errorf("UseLoad requires ConfigPath")
		}
		loaded, err := remoteconfig.Load(path)
		if err != nil {
			return nil, err
		}
		cfg = loaded
	} else if req.ConfigJSON != "" {
		var c remoteconfig.Config
		if err := json.Unmarshal([]byte(req.ConfigJSON), &c); err != nil {
			return nil, fmt.Errorf("parse ConfigJSON: %w", err)
		}
		cfg = &c
	}
	ep, state := remoteconfig.Resolve(cfg)
	resp.State = string(state)
	resp.Resolved = ep.OK
	resp.Server = ep.Server
	resp.Token = ep.Token
	return resp, nil
}

func runSave(t *testing.T, req *Request, resp *Response) (*Response, error) {
	dir, err := os.MkdirTemp("", "macos-remote-menubar-save-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	path := req.ConfigPath
	if path == "" {
		path = filepath.Join(dir, "remote-agent-config.json")
	}

	var cfg remoteconfig.Config
	if req.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(req.ConfigJSON), &cfg); err != nil {
			return nil, fmt.Errorf("parse ConfigJSON: %w", err)
		}
	}

	// Apply leaf mutations (domain token/server/default).
	if req.UpdateDefault != "" {
		cfg.Default = req.UpdateDefault
	}
	if req.UpdateServer != "" || req.UpdateToken != "" {
		if len(cfg.Domains) == 0 {
			cfg.Domains = []remoteconfig.Domain{{}}
		}
		if req.UpdateServer != "" {
			cfg.Domains[0].Server = req.UpdateServer
		}
		if req.UpdateToken != "" {
			cfg.Domains[0].Token = req.UpdateToken
		}
	}

	if err := remoteconfig.Save(path, &cfg); err != nil {
		return nil, err
	}
	resp.SavedOK = true

	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	resp.FileMode = st.Mode().Perm()

	// Re-load and report project_bindings as canonical JSON.
	loaded, err := remoteconfig.Load(path)
	if err != nil {
		return nil, err
	}
	if loaded == nil {
		return nil, fmt.Errorf("Load after Save returned nil")
	}
	bindJSON, err := json.Marshal(loaded.ProjectBindings)
	if err != nil {
		return nil, err
	}
	resp.ProjectBindingsJSON = string(bindJSON)
	return resp, nil
}

func runProfile(t *testing.T, req *Request, resp *Response) (*Response, error) {
	var p appprofile.Profile
	switch req.ProfileName {
	case "remote":
		p = appprofile.Remote()
	case "local":
		p = appprofile.Local()
	default:
		return nil, fmt.Errorf("unknown ProfileName %q", req.ProfileName)
	}
	resp.SpawnsDaemon = p.SpawnsDaemon
	resp.UsesAuthToken = p.UsesAuthToken
	resp.ConfigFileName = p.ConfigFileName
	resp.BundleID = p.BundleID
	resp.AppName = p.AppName
	resp.DisplayName = p.DisplayName
	return resp, nil
}

func runClientContract(t *testing.T, req *Request, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	switch req.ClientLeaf {
	case "no-restart-daemon":
		// Remote compliance (local app may still show Restart Daemon):
		// 1) dedicated remote AICriticApp without Restart Daemon, OR
		// 2) shared menu gates Restart Daemon on SpawnsDaemon / local profile.
		remoteApp := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-remote-macos", "AICriticApp.swift")
		sharedCandidates := []string{
			filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "AICriticApp.swift"),
			filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "AppProfile.swift"),
			filepath.Join(moduleRoot, "macos-ai-critic", "Shared", "AppProfile.swift"),
			filepath.Join(moduleRoot, "macos-ai-critic", "Shared", "AICriticApp.swift"),
			filepath.Join(moduleRoot, "macos-ai-critic", "Package.swift"),
		}

		if data, err := os.ReadFile(remoteApp); err == nil {
			resp.SourcesChecked = append(resp.SourcesChecked, remoteApp)
			remoteSrc := string(data)
			hasRemoteRestart := strings.Contains(remoteSrc, "Restart Daemon") ||
				strings.Contains(remoteSrc, "Restart Server")
			resp.HasRestartDaemonMenu = hasRemoteRestart
			resp.RestartDaemonGated = !hasRemoteRestart
			return resp, nil
		}

		var combined strings.Builder
		for _, p := range sharedCandidates {
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			resp.SourcesChecked = append(resp.SourcesChecked, p)
			combined.Write(data)
			combined.WriteByte('\n')
		}
		src := combined.String()
		if src == "" {
			// No sources — RED until product exists.
			return resp, nil
		}

		// Evidence that remote profile work started (shared dual-product).
		hasRemoteProfile := strings.Contains(src, "ai-critic-remote-macos") ||
			strings.Contains(src, "SpawnsDaemon") ||
			strings.Contains(src, "spawnsDaemon") ||
			strings.Contains(src, "AppProfile") ||
			strings.Contains(src, "isRemote")
		gated := regexp.MustCompile(`(?s)(SpawnsDaemon|spawnsDaemon|!isRemote|isRemote\s*==\s*false|profile\s*==\s*\.local|AppProfile\.(local|Local))[\s\S]{0,500}Restart Daemon`).MatchString(src) ||
			regexp.MustCompile(`(?s)if\s+[^{]{0,120}(SpawnsDaemon|spawnsDaemon)[\s\S]{0,300}Restart Daemon`).MatchString(src)

		// Without remote product/profile work, treat as not yet compliant (RED).
		if !hasRemoteProfile {
			resp.HasRestartDaemonMenu = true
			resp.RestartDaemonGated = false
			return resp, nil
		}
		resp.HasRestartDaemonMenu = !gated
		resp.RestartDaemonGated = gated
		return resp, nil

	case "install-remote-identity":
		installPath := filepath.Join(moduleRoot, "script", "macos-app", "install-remote.sh")
		bundlePath := filepath.Join(moduleRoot, "script", "macos-app", "bundle.sh")
		resp.SourcesChecked = []string{installPath, bundlePath}
		data, err := os.ReadFile(installPath)
		if err != nil {
			// Missing install-remote.sh → RED
			return resp, nil
		}
		src := string(data)
		// Prefer explicit assignments in install-remote.sh.
		resp.InstallAppName = firstMatch(src, `APP_NAME\s*=\s*"?([a-zA-Z0-9._-]+)"?`)
		if resp.InstallAppName == "" {
			// Also accept default expansion form: APP_NAME="${APP_NAME:-ai-critic-remote-macos}"
			resp.InstallAppName = firstMatch(src, `APP_NAME\s*=\s*"?\$\{APP_NAME:-([^}]+)\}"?`)
		}
		resp.InstallBundleID = firstMatch(src, `BUNDLE_ID\s*=\s*"?([a-zA-Z0-9._-]+)"?`)
		if resp.InstallBundleID == "" {
			resp.InstallBundleID = firstMatch(src, `BUNDLE_ID\s*=\s*"?\$\{BUNDLE_ID:-([^}]+)\}"?`)
		}
		// Embedding server binary is a local-app concern; remote install must not require it.
		embeds := strings.Contains(src, "ai-critic-server")
		if strings.Contains(src, "embed") && strings.Contains(src, "server binary") {
			embeds = true
		}
		// Explicit remote-only skip of server binary clears the flag.
		lower := strings.ToLower(src)
		if strings.Contains(lower, "no server binary") ||
			strings.Contains(src, "SKIP_SERVER") ||
			strings.Contains(src, "REMOTE_ONLY") ||
			strings.Contains(lower, "does not embed") {
			embeds = false
		}
		resp.EmbedsServerBinary = embeds
		return resp, nil

	default:
		return nil, fmt.Errorf("unknown ClientLeaf %q", req.ClientLeaf)
	}
}

func firstMatch(src, pattern string) string {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(src)
	if len(m) < 2 {
		return ""
	}
	return m[1]
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
