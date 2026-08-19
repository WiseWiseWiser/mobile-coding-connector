# macOS main window — Show App + sidebar pages

Pure formatters (`macosapp/mainwindow`) and Swift source contracts for the
local and remote menu-bar apps.

# DSN

**Participants**

- **Formatters** — sidebar titles, Show App label, last-page storage key.
- **Local app** — Show App, Window("AI Critic"), NavigationSplitView pages,
  Settings… opens settings page, no standalone Settings window.
- **Remote app** — same chrome; no iTerm switcher / no Open in iTerm2 on pages.

## Version

0.0.1

## Decision Tree

```
[macos-main-window]
 |
 +-- format/
 |    +-- sidebar-titles/
 |    +-- show-app-label/
 |    +-- normalize-unknown/
 |
 +-- client/
      +-- local-show-app/
      +-- remote-show-app/
      +-- local-main-window/
      +-- remote-main-window/
      +-- no-standalone-settings/
      +-- settings-opens-page/
      +-- remote-no-iterm-page/
```

## How to Run

```sh
doctest test ./tests/macos-main-window/...
```

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/mainwindow"
	"github.com/xhd2015/doctest/session"
)

type Request struct {
	Op         string // format | client
	SidebarID  string
	ClientLeaf string
}

type Response struct {
	Title         string
	ShowAppLabel  string
	Normalized    string
	StorageKey    string
	IDs           []string

	LocalHasShowApp      bool
	RemoteHasShowApp     bool
	LocalHasMainWindow   bool
	RemoteHasMainWindow  bool
	HasStandaloneSettings bool
	SettingsOpensPage    bool
	RemoteHasITermPage   bool
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "format":
		resp.Title = mainwindow.FormatSidebarTitle(req.SidebarID)
		resp.ShowAppLabel = mainwindow.FormatShowAppLabel()
		resp.Normalized = mainwindow.NormalizeSidebarID(req.SidebarID)
		resp.StorageKey = mainwindow.StorageKey
		resp.IDs = mainwindow.SidebarIDs()
		return resp, nil
	case "client":
		return runClient(d, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runClient(d *session.Doctest, req *Request, resp *Response) (*Response, error) {
	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	localDir := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos")
	remoteDir := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-remote-macos")
	sharedDir := filepath.Join(moduleRoot, "macos-ai-critic", "Shared")
	localSrc := readTree(localDir)
	remoteSrc := readTree(remoteDir)
	sharedSrc := readTree(sharedDir)
	localAll := localSrc + "\n" + sharedSrc
	remoteAll := remoteSrc + "\n" + sharedSrc

	resp.LocalHasShowApp = strings.Contains(localSrc, `show-app-menu-button`) ||
		strings.Contains(localSrc, `Button("Show App")`) ||
		strings.Contains(localSrc, "formatShowAppLabel")
	resp.RemoteHasShowApp = strings.Contains(remoteSrc, `show-app-menu-button`) ||
		strings.Contains(remoteSrc, `Button("Show App")`) ||
		strings.Contains(remoteSrc, "formatShowAppLabel")

	resp.LocalHasMainWindow = strings.Contains(localAll, `Window("AI Critic"`) &&
		strings.Contains(localAll, "NavigationSplitView") &&
		strings.Contains(localAll, `"Home"`) &&
		strings.Contains(sharedSrc, `"Services"`) &&
		strings.Contains(sharedSrc, `"Projects"`) &&
		strings.Contains(sharedSrc, `"Settings"`)
	resp.RemoteHasMainWindow = strings.Contains(remoteAll, `Window("AI Critic(Remote)"`) &&
		strings.Contains(remoteAll, "NavigationSplitView")

	resp.HasStandaloneSettings = strings.Contains(localSrc, `Window("Settings"`) ||
		strings.Contains(remoteSrc, `Window("Settings"`)
	resp.SettingsOpensPage = strings.Contains(localSrc, "open(page: .settings)") &&
		strings.Contains(remoteSrc, "open(page: .settings)")
	resp.RemoteHasITermPage = strings.Contains(remoteSrc, "onOpenInITerm") ||
		strings.Contains(remoteSrc, "openProjectInITerm2") ||
		strings.Contains(remoteSrc, "ITermSwitcher")
	return resp, nil
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
