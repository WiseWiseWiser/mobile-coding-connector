# macOS Menu Bar Terminals + Remote Domain Switcher Doctests

Pure-function tests for Terminals menu helpers (`macosapp/menubar`) and remote
domain default selection (`macosapp/remoteconfig`), plus Swift source contracts
for local (`ai-critic-macos`) and remote (`ai-critic-remote-macos`) menu-bar apps.

# DSN (Domain Specific Notion)

**Participants**

- **Terminal menu formatters (`macosapp/menubar`)** — pure helpers that format
  session titles, the empty-list placeholder, attach/new CLI command lines,
  agent binary names (`local-agent` vs `remote-agent`), and the periodic refresh
  interval for services + terminals.
- **Remote config package (`macosapp/remoteconfig`)** — load/save/resolve of
  `remote-agent-config.json`; domain switcher selects a domain and persists
  `default` so Services, Terminals, and Open-in-browser share one endpoint.
- **Local macOS menu bar (`ai-critic-macos`)** — Terminals submenu listing
  server sessions, New Terminal…, iTerm2 attach/new, top-level Refresh; **no**
  remote domain/Server switcher.
- **Remote macOS menu bar (`ai-critic-remote-macos`)** — same Terminals UX plus
  a **level-1** Server/domain switcher that writes `default` and reloads clients.
- **iTerm2 open path** — session click and New Terminal open **iTerm only**
  (no Terminal.app fallback). **Local app only** routes open through
  `POST /api/local/iterm2/open` (ServerClient) with mode + `send` (attach/new
  command); does not keep a parallel product-path raw osascript-only open.
  **Remote app** may keep client-side iTerm open (must not call open API on a
  remote server — that would run osascript remotely).
- **Test harness** — invokes Go helpers with leaf inputs or inspects Swift
  sources; no UI automation, no live iTerm, no network.

**Behaviors**

- `FormatTerminalTitle(name, id, status)`: base = non-empty trimmed name, else id;
  if trimmed status equals `exited` case-insensitively → base + ` [EXITED]`;
  else (running, empty, unknown) → base only. Cleared sessions are not listed by
  the API (no menubar title handling for cleared).
- `FormatTerminalsEmptyLabel()`: exact `No terminal sessions`.
- `BuildTerminalAttachCommand(agentBinary, sessionID)`:
  `{agentBinary} terminal attach {sessionID}` (space-joined; prefer session id).
- `BuildTerminalNewCommand(agentBinary)`: `{agentBinary} terminal new`.
- `AgentBinaryForApp(isRemote)`: remote → `remote-agent`; local → `local-agent`.
- `SelectDefaultDomain(cfg, serverURL)`: selected domain’s server becomes
  `default` (normalized match); Save then Resolve yields that endpoint.
- Periodic refresh interval: **30 seconds** (Go constant / helper).
- Swift: both apps expose Terminals + New Terminal…; remote has level-1 Server
  switcher; local does not; iTerm-only open path; top-level Refresh kept;
  periodic background refresh of services + terminals.
- Local Terminals attach/new: call `ServerClient.openITerm2` /
  `/api/local/iterm2/open` with appropriate `mode` + `send` (command line),
  not product-path-only `ITermOpener.openCommandOrAlert`.
- Remote Terminals: out of scope for open-API refactor; existing client-side
  open remains acceptable.

## Version

0.0.2

## Decision Tree

```
[macos-menubar-terminals]
 |
 +-- title/                              (GROUP)  session display title (+ status)
 |    +-- with-name/                     (LEAF)   non-empty name, running → base
 |    +-- empty-name/                    (LEAF)   empty name, empty status → id
 |    +-- whitespace-name/               (LEAF)   whitespace name, running → id
 |    +-- exited-with-name/              (LEAF)   name + status=exited → base [EXITED]
 |    +-- exited-empty-name/             (LEAF)   empty name + exited → id [EXITED]
 |    +-- exited-whitespace-name/        (LEAF)   whitespace name + exited → id [EXITED]
 |    +-- exited-case-insensitive/       (LEAF)   status ` Exited ` still suffixes
 |
 +-- empty/                              (GROUP)  empty Terminals list
 |    +-- label/                         (LEAF)   `No terminal sessions`
 |
 +-- command/                            (GROUP)  attach / new / agent binary
 |    +-- attach-local/                  (LEAF)   local-agent terminal attach
 |    +-- attach-remote/                 (LEAF)   remote-agent terminal attach
 |    +-- new-local/                     (LEAF)   local-agent terminal new
 |    +-- new-remote/                    (LEAF)   remote-agent terminal new
 |    +-- agent-binary-local/            (LEAF)   isRemote=false → local-agent
 |    +-- agent-binary-remote/           (LEAF)   isRemote=true → remote-agent
 |
 +-- domain/                             (GROUP)  remote default selection
 |    +-- select-persists-default/       (LEAF)   select B → default=B + resolve B
 |
 +-- interval/                           (GROUP)  periodic refresh constant
 |    +-- thirty-seconds/                (LEAF)   30s services+terminals poll
 |
 +-- client/                             (GROUP)  Swift source contracts
      +-- local-terminals-menu/          (LEAF)   local app Terminals menu
      +-- remote-terminals-menu/         (LEAF)   remote app Terminals menu
      +-- remote-server-switcher/        (LEAF)   remote level-1 Server switcher
      +-- local-no-domain-switcher/      (LEAF)   local has no domain switcher
      +-- iterm-only/                    (LEAF)   iTerm only; local open via API
      +-- open-via-local-api/            (LEAF)   local attach/new → open API
      +-- top-level-refresh/             (LEAF)   top-level Refresh retained
      +-- periodic-refresh/              (LEAF)   timer refreshes services+terms
      +-- new-terminal/                  (LEAF)   New Terminal… in both apps
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `title/with-name` | name=demo, id=abc, status=running → `demo` |
| 2 | `title/empty-name` | empty name, empty status → id `sess-1` |
| 3 | `title/whitespace-name` | whitespace name, status=running → `sess-1` |
| 4 | `title/exited-with-name` | name=demo, status=exited → `demo [EXITED]` |
| 5 | `title/exited-empty-name` | empty name, status=exited → `sess-1 [EXITED]` |
| 6 | `title/exited-whitespace-name` | whitespace name, status=exited → `sess-1 [EXITED]` |
| 7 | `title/exited-case-insensitive` | status=` Exited ` → still `demo [EXITED]` |
| 8 | `empty/label` | `FormatTerminalsEmptyLabel` → `No terminal sessions` |
| 9 | `command/attach-local` | `local-agent terminal attach web1` |
| 10 | `command/attach-remote` | `remote-agent terminal attach web1` |
| 11 | `command/new-local` | `local-agent terminal new` |
| 12 | `command/new-remote` | `remote-agent terminal new` |
| 13 | `command/agent-binary-local` | `AgentBinaryForApp(false)` → `local-agent` |
| 14 | `command/agent-binary-remote` | `AgentBinaryForApp(true)` → `remote-agent` |
| 15 | `domain/select-persists-default` | select domain B → `default` + resolve B |
| 16 | `interval/thirty-seconds` | periodic refresh interval is 30s |
| 17 | `client/local-terminals-menu` | local `AICriticApp.swift` has Terminals menu |
| 18 | `client/remote-terminals-menu` | remote `AICriticApp.swift` has Terminals menu |
| 19 | `client/remote-server-switcher` | remote level-1 Server/domain switcher |
| 20 | `client/local-no-domain-switcher` | local app has no domain switcher |
| 21 | `client/iterm-only` | iTerm only; no Terminal.app; local uses open API |
| 22 | `client/open-via-local-api` | Local attach/new → `/api/local/iterm2/open` |
| 23 | `client/top-level-refresh` | top-level Refresh button present |
| 24 | `client/periodic-refresh` | periodic services+terminals refresh present |
| 25 | `client/new-terminal` | New Terminal present in both apps |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| with-name | title | name=demo, id=abc, status=running | Title=`demo` |
| empty-name | title | name="", id=sess-1, status="" | Title=`sess-1` |
| whitespace-name | title | name=`"  \t  "`, id=sess-1, status=running | Title=`sess-1` |
| exited-with-name | title | name=demo, id=abc, status=exited | Title=`demo [EXITED]` |
| exited-empty-name | title | name="", id=sess-1, status=exited | Title=`sess-1 [EXITED]` |
| exited-whitespace-name | title | name=`"  \t  "`, id=sess-1, status=exited | Title=`sess-1 [EXITED]` |
| exited-case-insensitive | title | name=demo, id=abc, status=` Exited ` | Title=`demo [EXITED]` |
| label | empty | — | EmptyLabel=`No terminal sessions` |
| attach-local | attach_cmd | binary=local-agent, id=web1 | Command exact |
| attach-remote | attach_cmd | binary=remote-agent, id=web1 | Command exact |
| new-local | new_cmd | binary=local-agent | Command exact |
| new-remote | new_cmd | binary=remote-agent | Command exact |
| agent-binary-local | agent_binary | isRemote=false | `local-agent` |
| agent-binary-remote | agent_binary | isRemote=true | `remote-agent` |
| select-persists-default | select_domain | multi-domain, pick B | default+resolve B |
| thirty-seconds | interval | — | 30 seconds |
| local-terminals-menu | client | local Swift | Terminals menu |
| remote-terminals-menu | client | remote Swift | Terminals menu |
| remote-server-switcher | client | remote Swift | level-1 Server |
| local-no-domain-switcher | client | local Swift | no domain switcher |
| iterm-only | client | both apps | iTerm only + local API open |
| open-via-local-api | client | local app | attach/new via open API |
| top-level-refresh | client | both apps | Refresh button |
| periodic-refresh | client | both apps | timer + terminals |
| new-terminal | client | both apps | New Terminal |

## How to Run

```sh
doctest vet ./tests/macos-menubar-terminals
doctest test ./tests/macos-menubar-terminals/...
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

	"github.com/xhd2015/ai-critic/macosapp/menubar"
	"github.com/xhd2015/ai-critic/macosapp/remoteconfig"
)

type Request struct {
	Op string

	// title
	Name      string
	SessionID string
	Status    string

	// command builders
	AgentBinary string
	IsRemote    bool

	// domain select
	ConfigJSON   string
	SelectServer string

	// client source contracts
	ClientLeaf string
}

type Response struct {
	Title      string
	EmptyLabel string
	Command    string
	AgentBinary string

	// domain
	DefaultServer  string
	ResolvedServer string
	ResolvedToken  string
	ResolvedOK     bool
	State          string
	SavedOK        bool

	// interval
	RefreshIntervalSec int

	// client contract flags
	HasLocalTerminalsMenu   bool
	HasRemoteTerminalsMenu  bool
	HasRemoteServerSwitcher bool
	LocalHasDomainSwitcher  bool
	UsesITermOnly           bool
	HasTerminalAppFallback  bool
	// Local product path uses POST /api/local/iterm2/open (not raw osascript-only).
	OpensViaLocalITerm2API  bool
	// Local attach/new call openITerm2 / open API with send/mode.
	LocalTerminalsUseOpenAPI bool
	// Product paths still calling ITermOpener.openCommandOrAlert for terminals.
	HasDirectITermOpenerProductPath bool
	HasTopLevelRefresh      bool
	HasPeriodicRefresh      bool
	HasNewTerminal          bool
	SwiftSourcesChecked     []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "title":
		resp.Title = menubar.FormatTerminalTitle(req.Name, req.SessionID, req.Status)
	case "empty":
		resp.EmptyLabel = menubar.FormatTerminalsEmptyLabel()
	case "attach_cmd":
		resp.Command = menubar.BuildTerminalAttachCommand(req.AgentBinary, req.SessionID)
	case "new_cmd":
		resp.Command = menubar.BuildTerminalNewCommand(req.AgentBinary)
	case "agent_binary":
		resp.AgentBinary = menubar.AgentBinaryForApp(req.IsRemote)
	case "select_domain":
		return runSelectDomain(t, req, resp)
	case "interval":
		d := menubar.PeriodicRefreshInterval
		resp.RefreshIntervalSec = int(d / time.Second)
	case "client":
		return runClientContract(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
	return resp, nil
}

func runSelectDomain(t *testing.T, req *Request, resp *Response) (*Response, error) {
	var cfg remoteconfig.Config
	if err := json.Unmarshal([]byte(req.ConfigJSON), &cfg); err != nil {
		return nil, fmt.Errorf("parse ConfigJSON: %w", err)
	}

	updated, err := remoteconfig.SelectDefaultDomain(&cfg, req.SelectServer)
	if err != nil {
		return nil, err
	}

	dir, err := os.MkdirTemp("", "macos-menubar-terminals-domain-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "remote-agent-config.json")

	if err := remoteconfig.Save(path, updated); err != nil {
		return nil, err
	}
	resp.SavedOK = true

	loaded, err := remoteconfig.Load(path)
	if err != nil {
		return nil, err
	}
	if loaded == nil {
		return nil, fmt.Errorf("Load after Save returned nil")
	}
	resp.DefaultServer = loaded.Default

	ep, state := remoteconfig.Resolve(loaded)
	resp.State = string(state)
	resp.ResolvedOK = ep.OK
	resp.ResolvedServer = ep.Server
	resp.ResolvedToken = ep.Token
	return resp, nil
}

func runClientContract(t *testing.T, req *Request, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}
	localApp := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "AICriticApp.swift")
	remoteApp := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-remote-macos", "AICriticApp.swift")
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

	// Terminals menus (Menu("Terminals") or equivalent string + structure)
	resp.HasLocalTerminalsMenu = hasTerminalsMenu(localStr)
	resp.HasRemoteTerminalsMenu = hasTerminalsMenu(remoteStr)

	// Level-1 Server / domain switcher on remote only
	resp.HasRemoteServerSwitcher = hasLevel1ServerSwitcher(remoteStr)
	resp.LocalHasDomainSwitcher = hasLevel1ServerSwitcher(localStr)

	// iTerm only — open path must reference iTerm; must not fall back to Terminal.app
	resp.UsesITermOnly = referencesITerm(both) && !hasTerminalAppFallback(both)
	resp.HasTerminalAppFallback = hasTerminalAppFallback(both)

	// Local open via server API (REQUIREMENT-DESIGN-local-iterm2-open)
	localAndServer := localStr + "\n"
	serverClientPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "ServerClient.swift")
	if sc, err := os.ReadFile(serverClientPath); err == nil {
		localAndServer += string(sc)
		resp.SwiftSourcesChecked = append(resp.SwiftSourcesChecked, serverClientPath)
	}
	resp.OpensViaLocalITerm2API = strings.Contains(localAndServer, "/api/local/iterm2/open")
	resp.LocalTerminalsUseOpenAPI = resp.OpensViaLocalITerm2API &&
		(strings.Contains(localStr, "openITerm2") ||
			strings.Contains(localAndServer, "openITerm2") ||
			strings.Contains(localStr, "openIterm2")) &&
		(strings.Contains(localStr, "openAttachTerminal") || strings.Contains(localStr, "openNewTerminal") ||
			strings.Contains(localStr, "terminal attach") || strings.Contains(localStr, "terminal new") ||
			strings.Contains(localStr, "BuildTerminalAttach") || strings.Contains(localStr, "buildTerminalAttach") ||
			strings.Contains(localStr, "BuildTerminalNew") || strings.Contains(localStr, "buildTerminalNew"))
	// Product-path direct osascript open for terminals (to be retired on local app).
	resp.HasDirectITermOpenerProductPath = strings.Contains(localStr, "ITermOpener.openCommandOrAlert") ||
		strings.Contains(localStr, "ITermOpener.openCommand(")

	// Top-level Refresh retained
	resp.HasTopLevelRefresh = hasTopLevelRefresh(localStr) && hasTopLevelRefresh(remoteStr)

	// Periodic refresh of services + terminals
	resp.HasPeriodicRefresh = hasPeriodicTerminalsRefresh(localStr) || hasPeriodicTerminalsRefresh(remoteStr) ||
		hasPeriodicTerminalsRefresh(sharedCombined)

	// New Terminal present in both apps
	resp.HasNewTerminal = hasNewTerminal(localStr) && hasNewTerminal(remoteStr)

	switch req.ClientLeaf {
	case "local-terminals-menu",
		"remote-terminals-menu",
		"remote-server-switcher",
		"local-no-domain-switcher",
		"iterm-only",
		"open-via-local-api",
		"top-level-refresh",
		"periodic-refresh",
		"new-terminal":
		// fields populated above
	default:
		return nil, fmt.Errorf("unknown client leaf %q", req.ClientLeaf)
	}
	return resp, nil
}

func hasTerminalsMenu(src string) bool {
	if strings.Contains(src, `Menu("Terminals")`) || strings.Contains(src, `Menu("Terminals…")`) {
		return true
	}
	// SwiftUI accessibility or title variants
	return regexp.MustCompile(`Menu\s*\(\s*"Terminals`).MatchString(src) ||
		strings.Contains(src, "terminals-menu") ||
		regexp.MustCompile(`(?i)Menu\s*\{\s*//\s*Terminals`).MatchString(src)
}

func hasLevel1ServerSwitcher(src string) bool {
	// Level-1 menu for domain/server selection (not nested only under Terminals)
	if regexp.MustCompile(`Menu\s*\(\s*"Server`).MatchString(src) {
		return true
	}
	if regexp.MustCompile(`Menu\s*\(\s*"Servers`).MatchString(src) {
		return true
	}
	if strings.Contains(src, "server-switcher") || strings.Contains(src, "domain-switcher") {
		return true
	}
	// Selecting a domain writes default — look for SelectDefault / setDefault patterns
	// combined with Menu at root body level.
	return regexp.MustCompile(`(?i)(selectDefault|setDefault|defaultDomain|switchDomain)`).MatchString(src) &&
		regexp.MustCompile(`Menu\s*\(`).MatchString(src)
}

func referencesITerm(src string) bool {
	return strings.Contains(src, "iTerm") ||
		strings.Contains(src, "iTerm2") ||
		strings.Contains(src, "iTerm.app") ||
		strings.Contains(src, "/Applications/iTerm.app")
}

func hasTerminalAppFallback(src string) bool {
	// Fallback path that opens Apple Terminal.app when iTerm is missing.
	// Presence of bare "Terminal" in UI copy (e.g. "New Terminal") is OK.
	if strings.Contains(src, "/Applications/Terminal.app") {
		return true
	}
	if regexp.MustCompile(`(?i)fallback[\s\S]{0,80}Terminal\.app`).MatchString(src) {
		return true
	}
	if regexp.MustCompile(`(?i)Terminal\.app[\s\S]{0,80}fallback`).MatchString(src) {
		return true
	}
	// NSWorkspace open of Terminal as alternate open path
	return regexp.MustCompile(`(?i)open.*Terminal\.app|Terminal\.app.*open`).MatchString(src) &&
		!strings.Contains(src, "iTerm")
}

func hasTopLevelRefresh(src string) bool {
	return strings.Contains(src, `Button("Refresh")`) ||
		regexp.MustCompile(`Button\s*\(\s*"Refresh"`).MatchString(src)
}

func hasNewTerminal(src string) bool {
	return strings.Contains(src, "New Terminal") ||
		regexp.MustCompile(`(?i)New Terminal`).MatchString(src)
}

func hasPeriodicTerminalsRefresh(src string) bool {
	// Background loop / timer that re-fetches terminals (and ideally services)
	hasSleepOrTimer := strings.Contains(src, "Task.sleep") ||
		strings.Contains(src, "Timer") ||
		strings.Contains(src, "nanoseconds:") ||
		regexp.MustCompile(`startRefresh|refreshLoop|periodicRefresh`).MatchString(src)
	hasTerminalsFetch := regexp.MustCompile(`(?i)(refreshTerminals|listTerminal|terminal/sessions|TerminalSession)`).MatchString(src)
	// Accept combined refresh that mentions terminals alongside services
	hasCombined := hasSleepOrTimer && (hasTerminalsFetch ||
		(strings.Contains(src, "refreshServices") && hasTerminalsFetch))
	return hasCombined || (hasSleepOrTimer && hasTerminalsFetch)
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
