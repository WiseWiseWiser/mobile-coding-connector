# macOS Menu Bar Services Formatting Doctests

Pure-function tests for `macosapp/menubar` service menu formatters — the Go spec
mirrored by the Swift `ai-critic-macos` client when rendering the Services submenu.
Swift contract leaves read sources under `macos-ai-critic/ai-critic-macos/`.

# DSN (Domain Specific Notion)

**Participants**

- **Service menu formatters (`macosapp/menubar`)** — map `ServiceStatus` fields to
  submenu titles, action gating, and disable/enable alert copy aligned with
  `server/services` constants.
- **macOS menu bar (`AICriticApp.swift`)** — nested SwiftUI `Menu` per service with
  Start, Stop, Restart, Disable/Enable, and View Logs actions.
- **Swift `ServerClient`** — HTTP client on main server port `23712` for
  `GET /api/services?all=1`, grok/codex usage, and service actions.
- **Swift `DaemonClient`** — fallback on keep-alive port `23312` only for control-plane
  ops (status, restart daemon) when server is unreachable.
- **Test harness** — invokes Go formatters with leaf inputs or inspects Swift sources;
  no UI rendering.

**Behaviors**

- `FormatServiceTitle`: running → `{name} ● Running`; stopped+disabled →
  `{name} ○ Stopped (disabled)`; error → `{name} ⚠ Error` (full name + error presentation).
- `CanStopService`: false when `pid==0 && !desiredRunning`; true when `pid>0`.
- `ShowEnableAction`: disabled service shows **Enable** (not Disable).
- `DisableAlertMessage` / `EnableAlertMessage`: mirror `msgDisableRunning` and
  `msgEnableStopped` from `server/services`.
- `FormatServicesEmptyLabel`: `No services configured`.
- Swift contract: business APIs use `ServerClient` on `23712`; nested per-service
  `Menu`; View Logs streams via `GET /api/logs/stream?path=...&lines=1000` (server
  SSE, not local `tail`).

## Version

0.0.2

## Decision Tree

```
[menubar services]
 |
 +-- title/                           (GROUP)  service submenu title strings
 |    +-- running/                    (LEAF)   `web ● Running`
 |    +-- stopped-disabled/           (LEAF)   `api ○ Stopped (disabled)`
 |    +-- error/                      (LEAF)   `web ⚠ Error`
 |
 +-- action/                          (GROUP)  per-service action gating
 |    +-- stop-disabled/             (LEAF)   canStop=false when pid=0 && !desired
 |    +-- stop-enabled/              (LEAF)   canStop=true when pid>0
 |    +-- toggle-enable/             (LEAF)   disabled → Enable not Disable
 |
 +-- alert/                           (GROUP)  disable/enable NSAlert copy
 |    +-- disable-running/           (LEAF)   matches msgDisableRunning
 |    +-- enable-stopped/            (LEAF)   matches msgEnableStopped
 |
 +-- dropdown/                        (GROUP)  services menu root
 |    +-- empty/                      (LEAF)   `No services configured`
 |
 +-- client/                          (GROUP)  Swift source contract
      +-- swift-server-port/         (LEAF)   grok/codex/services on server port
      +-- swift-services-submenu/    (LEAF)   nested Menu per service
      +-- swift-log-stream/          (LEAF)   server SSE log stream
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `title/running` | `FormatServiceTitle` running → `web ● Running` |
| 2 | `title/stopped-disabled` | stopped+disabled → `api ○ Stopped (disabled)` |
| 3 | `title/error` | error status → `web ⚠ Error` |
| 4 | `action/stop-disabled` | `CanStopService` false when pid=0 && !desiredRunning |
| 5 | `action/stop-enabled` | `CanStopService` true when pid>0 |
| 6 | `action/toggle-enable` | `ShowEnableAction` true when disabled |
| 7 | `alert/disable-running` | `DisableAlertMessage(true)` → msgDisableRunning |
| 8 | `alert/enable-stopped` | `EnableAlertMessage(false)` → msgEnableStopped |
| 9 | `dropdown/empty` | `FormatServicesEmptyLabel` → `No services configured` |
| 10 | `client/swift-server-port` | Swift uses ServerClient:23712 for business APIs |
| 11 | `client/swift-services-submenu` | Nested Menu for per-service actions |
| 12 | `client/swift-log-stream` | View Logs streams `/api/logs/stream` SSE |

## Parameter Coverage

| Leaf | Op | Key inputs | Expected |
|------|-----|------------|----------|
| running | title | name=web, status=running, enabled=true | `web ● Running` |
| stopped-disabled | title | name=api, status=stopped, enabled=false | `api ○ Stopped (disabled)` |
| error | title | name=web, status=error | `web ⚠ Error` |
| stop-disabled | action | pid=0, desiredRunning=false | canStop=false |
| stop-enabled | action | pid=1234 | canStop=true |
| toggle-enable | action | enabled=false | showEnable=true |
| disable-running | alert | running=true | msgDisableRunning |
| enable-stopped | alert | running=false | msgEnableStopped |
| empty | empty | — | `No services configured` |
| swift-server-port | client | Swift sources | port 23712, not 23312 |
| swift-services-submenu | client | AICriticApp.swift | nested Menu |
| swift-log-stream | client | ServerClient + LogTailWindow | `/api/logs/stream` SSE, lines=1000 |

## How to Run

```sh
doctest vet ./tests/macos-menubar-services
doctest test ./tests/macos-menubar-services/...
```

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
	"github.com/xhd2015/ai-critic/server/config"
)

const (
	msgDisableRunning = "The server won't stop immediately unless you manually stop it"
	msgEnableStopped  = "The server won't start immediately until daemon checks at next time"
)

type Request struct {
	Op string

	// title / action / alert / empty
	Name           string
	Status         string
	Enabled        bool
	PID            int
	DesiredRunning bool
	Running        bool // alert: service process alive at action time

	// client
	ClientLeaf string
}

type Response struct {
	Title        string
	CanStop      bool
	ShowEnable   bool
	AlertMessage string
	EmptyLabel   string

	// client contract
	ServerPort              int
	GrokUsesServerPort      bool
	CodexUsesServerPort     bool
	ServicesUsesAllQuery    bool
	DaemonPortForGrok       bool
	HasNestedServiceMenu    bool
	HasLogStreamEndpoint    bool
	ServerClientNoLocalTail bool
	LogWindowUsesStream     bool
	LogStreamLines1000      bool
	ViewLogsInvokesStream   bool
	SwiftSourcesChecked     []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "title":
		resp.Title = menubar.FormatServiceTitle(req.Name, req.Status, req.Enabled)
	case "action":
		resp.CanStop = menubar.CanStopService(req.PID, req.DesiredRunning)
		resp.ShowEnable = menubar.ShowEnableAction(req.Enabled)
	case "alert":
		if req.Running {
			resp.AlertMessage = menubar.DisableAlertMessage(req.Running)
		} else {
			resp.AlertMessage = menubar.EnableAlertMessage(req.Running)
		}
	case "empty":
		resp.EmptyLabel = menubar.FormatServicesEmptyLabel()
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
	daemonClientPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "DaemonClient.swift")
	logTailWindowPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "LogTailWindow.swift")
	resp.SwiftSourcesChecked = []string{appPath, serverClientPath, daemonClientPath, logTailWindowPath}

	appSrc, err := os.ReadFile(appPath)
	if err != nil {
		return nil, fmt.Errorf("read AICriticApp.swift: %w", err)
	}
	serverSrc, err := os.ReadFile(serverClientPath)
	if err != nil {
		return nil, fmt.Errorf("read ServerClient.swift: %w", err)
	}
	daemonSrc, err := os.ReadFile(daemonClientPath)
	if err != nil {
		return nil, fmt.Errorf("read DaemonClient.swift: %w", err)
	}
	logTailSrc, err := os.ReadFile(logTailWindowPath)
	if err != nil {
		return nil, fmt.Errorf("read LogTailWindow.swift: %w", err)
	}
	appStr := string(appSrc)
	serverStr := string(serverSrc)
	logTailStr := string(logTailSrc)
	combined := appStr + "\n" + serverStr + "\n" + string(daemonSrc) + "\n" + logTailStr

	resp.ServerPort = config.DefaultServerPort
	resp.GrokUsesServerPort = strings.Contains(serverStr, "/api/grok/usage") &&
		strings.Contains(serverStr, "23712")
	resp.CodexUsesServerPort = strings.Contains(serverStr, "/api/codex/usage") &&
		strings.Contains(serverStr, "23712")
	resp.ServicesUsesAllQuery = strings.Contains(combined, "/api/services") &&
		strings.Contains(combined, "all=1")
	resp.DaemonPortForGrok = regexp.MustCompile(`DaemonClient[\s\S]{0,400}/api/grok/usage`).MatchString(combined)

	switch req.ClientLeaf {
	case "swift-server-port":
		// fields set above
	case "swift-services-submenu":
		resp.HasNestedServiceMenu = regexp.MustCompile(`Menu\s*\{[\s\S]*Button\("Start"`).MatchString(appStr) ||
			regexp.MustCompile(`ForEach[\s\S]*Menu\s*\(`).MatchString(appStr)
	case "swift-log-stream":
		resp.HasLogStreamEndpoint = strings.Contains(serverStr, "/api/logs/stream")
		resp.ServerClientNoLocalTail = !strings.Contains(serverStr, "/usr/bin/tail") &&
			!regexp.MustCompile(`Process\s*\(`).MatchString(serverStr)
		usesStream := strings.Contains(logTailStr, "URLSession") ||
			strings.Contains(logTailStr, "text/event-stream") ||
			strings.Contains(strings.ToLower(logTailStr), "sse") ||
			strings.Contains(logTailStr, "streamLog")
		noLocalTail := !strings.Contains(logTailStr, "/usr/bin/tail") &&
			!regexp.MustCompile(`executableURL`).MatchString(logTailStr)
		resp.LogWindowUsesStream = usesStream && noLocalTail
		streamSrc := serverStr + "\n" + logTailStr
		resp.LogStreamLines1000 = strings.Contains(streamSrc, "lines=1000") ||
			strings.Contains(streamSrc, "lines: 1000") ||
			strings.Contains(streamSrc, "&lines=1000") ||
			regexp.MustCompile(`lines[^\n]{0,60}1000`).MatchString(streamSrc)
		resp.ViewLogsInvokesStream = strings.Contains(appStr, "View Logs") &&
			!strings.Contains(appStr, "/usr/bin/tail") &&
			(strings.Contains(appStr, "streamLog") ||
				strings.Contains(appStr, "LogTailWindow") ||
				strings.Contains(appStr, "/api/logs/stream"))
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