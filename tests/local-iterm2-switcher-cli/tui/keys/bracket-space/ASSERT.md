## Expected

1. After `]`, sidebar filter is a Desktop (not All) — SidebarFilterID or view tokens.
2. List is refiltered: not both `grok review` and `new tab` when only one Desktop is selected.
3. Focus stays on list (] does not switch pane).

```go
import (
	"regexp"
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ViewText == "" {
		t.Fatal("View empty after ]")
	}
	if resp.FocusPane != "list" {
		t.Fatalf("] must keep list focus; FocusPane=%q", resp.FocusPane)
	}
	// Filter advanced off All
	if resp.SidebarFilterID == "all" || resp.SidebarFilterID == "" {
		// allow detection via view if IDs wired: must show Desktop selection
		if !regexp.MustCompile(`Desktop [12]`).MatchString(resp.ViewText) {
			t.Fatalf("] should select a Desktop filter; SidebarFilterID=%q view=%q",
				resp.SidebarFilterID, resp.ViewText)
		}
		if resp.SidebarFilterID == "all" {
			t.Fatal("SidebarFilterID still all after ]")
		}
	}
	if strings.Contains(resp.SidebarFilterID, "desktop") || regexp.MustCompile(`desktop:`).MatchString(resp.SidebarFilterID) {
		// good
	} else if resp.SidebarFilterID != "" && resp.SidebarFilterID != "all" {
		// bookmarks etc. unexpected for first ]
		t.Fatalf("SidebarFilterID=%q want desktop:* after ] from All", resp.SidebarFilterID)
	} else if resp.SidebarFilterID == "" {
		// implementer may only reflect in View; require not both sessions when filtered
	}
	hasA := regexp.MustCompile(`grok review`).MatchString(resp.ViewText)
	hasB := regexp.MustCompile(`new tab`).MatchString(resp.ViewText)
	if hasA && hasB {
		t.Fatalf("] refilter: must not show both spaces' sessions; view=%q", resp.ViewText)
	}
	if !hasA && !hasB {
		t.Fatalf("] refilter: want one Desktop's sessions; view=%q", resp.ViewText)
	}
}
```
