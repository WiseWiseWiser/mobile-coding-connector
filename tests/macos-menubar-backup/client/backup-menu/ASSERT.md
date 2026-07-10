## Expected

1. `HasBackupMenu` is true.
2. `HasBackupNowItem` is true (`Backup Now` copy present).
3. `HasRevealInFinderItem` is true (`Reveal in Finder` copy present).

## Side Effects

- None (read-only source inspection).

## Errors

- Backup submenu missing from remote app; or only CLI without menu entry.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasBackupMenu {
		t.Fatalf("remote app missing Backup menu (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.HasBackupNowItem {
		t.Fatal("missing Backup Now… menu item copy")
	}
	if !resp.HasRevealInFinderItem {
		t.Fatal("missing Reveal in Finder… menu item copy")
	}
}
```
