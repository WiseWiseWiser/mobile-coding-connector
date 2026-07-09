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
- **iTerm2 opener** — session click and New Terminal open **iTerm only**
  (no Terminal.app fallback).
- **Test harness** — invokes Go helpers with leaf inputs or inspects Swift
  sources; no UI automation, no live iTerm, no network.

**Behaviors**

- `FormatTerminalTitle(name, id)`: if `strings.TrimSpace(name) != ""` → name;
  else → id.
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

## Version

0.0.2

## Decision Tree

```
[macos-menubar-terminals]
 |
 +-- title/                              (GROUP)  session display title
 |    +-- with-name/                     (LEAF)   non-empty name wins
 |    +-- empty-name/                    (LEAF)   empty name → id
 |    +-- whitespace-name/               (LEAF)   whitespace-only name → id
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
      +-- iterm-only/                    (LEAF)   iTerm only; no Terminal.app
      +-- top-level-refresh/             (LEAF)   top-level Refresh retained
      +-- periodic-refresh/              (LEAF)   timer refreshes services+terms
      +-- new-terminal/                  (LEAF)   New Terminal… in both apps
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `title/with-name` | `FormatTerminalTitle("demo","abc")` → `demo` |
| 2 | `title/empty-name` | empty name → id `sess-1` |
| 3 | `title/whitespace-name` | whitespace name → id `sess-1` |
| 4 | `empty/label` | `FormatTerminalsEmptyLabel` → `No terminal sessions` |
| 5 | `command/attach-local` | `local-agent terminal attach web1` |
| 6 | `command/attach-remote` | `remote-agent terminal attach web1` |
| 7 | `command/new-local` | `local-agent terminal new` |
| 8 | `command/new-remote` | `remote-agent terminal new` |
| 9 | `command/agent-binary-local` | `AgentBinaryForApp(false)` → `local-agent` |
| 10 | `command/agent-binary-remote` | `AgentBinaryForApp(true)` → `remote-agent` |
| 11 | `domain/select-persists-default` | select domain B → `default` + resolve B |
| 12 | `interval/thirty-seconds` | periodic refresh interval is 30s |
| 13 | `client/local-terminals-menu` | local `AICriticApp.swift` has Terminals menu |
| 14 | `client/remote-terminals-menu` | remote `AICriticApp.swift` has Terminals menu |
| 15 | `client/remote-server-switcher` | remote level-1 Server/domain switcher |
| 16 | `client/local-no-domain-switcher` | local app has no domain switcher |
| 17 | `client/iterm-only` | iTerm open path; no Terminal.app fallback |
| 18 | `client/top-level-refresh` | top-level Refresh button present |
| 19 | `client/periodic-refresh` | periodic services+terminals refresh present |
| 20 | `client/new-terminal` | New Terminal present in both apps |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| with-name | title | name=demo, id=abc | Title=`demo` |
| empty-name | title | name="", id=sess-1 | Title=`sess-1` |
| whitespace-name | title | name=`"  \t  "`, id=sess-1 | Title=`sess-1` |
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
| iterm-only | client | both apps | iTerm only |
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
	HasTopLevelRefresh      bool
	HasPeriodicRefresh      bool
	HasNewTerminal          bool
	SwiftSourcesChecked     []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "title":
		resp.Title = menubar.FormatTerminalTitle(req.Name, req.SessionID)
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
