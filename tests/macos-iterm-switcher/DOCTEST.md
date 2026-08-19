# macOS iTerm Switcher — formatters and local-app contracts

Pure formatters in `macosapp/itermswitcher` (Swift mirrors in Shared).
Local app wires hotkey + panel + ServerClient; remote app must not.

# DSN (Domain Specific Notion)

**Participants**

- **Formatters** — session title, desktop header, saved-notes header, match query.
- **Local app** — iTerm Switcher menu, inventory/focus/notes client paths.
- **Remote app** — no switcher UI.
- **Shared models** — `ITermLiveSession` decodes inventory JSON.

**Behaviors**

- Local app calls inventory/stream, focus, and notes APIs; remote app has no switcher.
- Shared `ITermLiveSession` decodes `agent_runner` and `grok_session_id`.

## Version

0.0.1

## Decision Tree

```
[macos-iterm-switcher]
 |
 +-- format/
 |    +-- title-with-note/
 |    +-- empty-name-cwd/
 |    +-- empty-name-id/
 |    +-- desktop-header/
 |    +-- saved-notes-header/
 |    +-- match-query/
 |    +-- sidebar-all/
 |    +-- sidebar-bookmarks/
 |    +-- sidebar-desktop/
 |    +-- filter-bookmarks/
 |    +-- filter-desktop/
 |    +-- filter-query/
 |
 +-- client/
      +-- local-has-switcher/
      +-- remote-no-switcher/
      +-- inventory-api/
      +-- focus-api/
      +-- notes-api/
      +-- click-outside-dismiss/
      +-- split-layout/
      +-- live-session-agent-fields/
```

## How to Run

```sh
doctest vet ./tests/macos-iterm-switcher
doctest test ./tests/macos-iterm-switcher/...
```

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/itermswitcher"
	"github.com/xhd2015/doctest/session"
)

type Request struct {
	Op string // format | client

	Name      string
	Note      string
	Cwd       string
	SessionID string
	SpaceIndex int
	Query     string
	SavedN    int
	SidebarID string
	Bookmarked bool

	ClientLeaf string
}

type Response struct {
	Title          string
	Note           string
	DesktopHeader  string
	SavedHeader    string
	Matches        bool
	DefaultHotKey  string
	SidebarTitle   string
	FilteredCount  int
	WindowTitle    string

	LocalHasSwitcher  bool
	RemoteHasSwitcher bool
	HasInventoryAPI   bool
	HasFocusAPI       bool
	HasNotesAPI       bool
	HasClickOutsideDismiss bool
	HasNativeSplit    bool
	HasBookmarkAction bool
	HasDoubleClickFocus bool
	HasLiveSessionAgentFields bool
	SwiftSources      []string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "format":
		resp.Title = itermswitcher.FormatSessionTitle(req.Name, req.Note, req.Cwd, req.SessionID)
		resp.Note = itermswitcher.FormatSessionNote(req.Note)
		resp.DesktopHeader = itermswitcher.FormatDesktopHeader(req.SpaceIndex)
		resp.SavedHeader = itermswitcher.FormatSavedNotesHeader(req.SavedN)
		resp.Matches = itermswitcher.SessionMatches(req.Name, req.Note, req.Cwd, "", "", req.SessionID, req.SpaceIndex, req.Query)
		resp.DefaultHotKey = itermswitcher.FormatDefaultHotKey()
		resp.SidebarTitle = itermswitcher.FormatSidebarTitle(req.SidebarID)
		resp.WindowTitle = itermswitcher.FormatWindowTitle()
		sessions := []itermswitcher.FilterSession{
			{SessionID: "a", Name: "grok review", Note: "fix auth", SpaceIndex: 0, Bookmarked: true},
			{SessionID: "b", Name: "wrk build", SpaceIndex: 1, Bookmarked: false},
			{SessionID: "c", Name: "logs", Note: "tail prod", SpaceIndex: 0, Bookmarked: true},
		}
		resp.FilteredCount = len(itermswitcher.FilterSessions(sessions, req.SidebarID, req.Query))
		return resp, nil
	case "client":
		return runClient(t, d, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runClient(t *testing.T, d *session.Doctest, req *Request, resp *Response) (*Response, error) {
	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	localApp := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "AICriticApp.swift")
	remoteApp := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-remote-macos", "AICriticApp.swift")
	localDir := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos")
	remoteDir := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-remote-macos")
	sharedDir := filepath.Join(moduleRoot, "macos-ai-critic", "Shared")

	localSrc := readTree(localDir)
	remoteSrc := readTree(remoteDir)
	sharedSrc := readTree(sharedDir)
	resp.SwiftSources = []string{localApp, remoteApp, sharedDir}

	localAll := localSrc + "\n" + sharedSrc
	remoteAll := remoteSrc + "\n" + sharedSrc

	resp.LocalHasSwitcher = strings.Contains(localAll, "iTerm Switcher") ||
		strings.Contains(localAll, "ITermSwitcher")
	resp.RemoteHasSwitcher = strings.Contains(remoteSrc, "iTerm Switcher") ||
		strings.Contains(remoteSrc, "ITermSwitcherController") ||
		strings.Contains(remoteSrc, "toggleSwitcher")

	resp.HasInventoryAPI = strings.Contains(localAll, "/api/local/iterm2/inventory/stream")
	resp.HasFocusAPI = strings.Contains(localAll, "/api/local/iterm2/focus")
	resp.HasNotesAPI = strings.Contains(localAll, "/api/local/iterm2/notes")
	resp.HasClickOutsideDismiss = strings.Contains(localSrc, "addGlobalMonitorForEvents") &&
		strings.Contains(localSrc, "addLocalMonitorForEvents") &&
		strings.Contains(localSrc, "dismissIfClickOutside")
	resp.HasNativeSplit = strings.Contains(localSrc, "NavigationSplitView") &&
		strings.Contains(localSrc, "searchable")
	resp.HasBookmarkAction = strings.Contains(localSrc, "toggleBookmark") &&
		strings.Contains(localAll, "bookmarked") &&
		strings.Contains(localSrc, "buttonStyle(.borderless)")
	resp.HasDoubleClickFocus = strings.Contains(localSrc, "clickCount") &&
		strings.Contains(localSrc, "ITermFocusHook")
	resp.HasLiveSessionAgentFields = swiftLiveSessionHasAgentFields(sharedSrc)
	_ = remoteAll
	return resp, nil
}

func swiftLiveSessionHasAgentFields(sharedSrc string) bool {
	idx := strings.Index(sharedSrc, "struct ITermLiveSession")
	if idx < 0 {
		return false
	}
	rest := sharedSrc[idx:]
	if end := strings.Index(rest[1:], "\npublic struct "); end > 0 {
		rest = rest[:end+1]
	}
	return strings.Contains(rest, "agent_runner") && strings.Contains(rest, "grok_session_id")
}

func readTree(dir string) string {
	var b strings.Builder
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".swift") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		b.Write(data)
		b.WriteByte('\n')
		return nil
	})
	return b.String()
}
```
