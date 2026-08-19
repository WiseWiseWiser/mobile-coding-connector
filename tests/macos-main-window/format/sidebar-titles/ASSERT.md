```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
	"github.com/xhd2015/ai-critic/macosapp/mainwindow"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "Home" {
		t.Fatalf("title=%q", resp.Title)
	}
	want := []string{"home", "services", "projects", "settings"}
	if len(resp.IDs) != len(want) {
		t.Fatalf("ids=%v", resp.IDs)
	}
	for i, id := range want {
		if resp.IDs[i] != id {
			t.Fatalf("ids[%d]=%q want %q", i, resp.IDs[i], id)
		}
		if mainwindow.FormatSidebarTitle(id) == "" {
			t.Fatalf("empty title for %s", id)
		}
	}
	if resp.StorageKey != "mainSidebarPage" {
		t.Fatalf("key=%q", resp.StorageKey)
	}
}
```
