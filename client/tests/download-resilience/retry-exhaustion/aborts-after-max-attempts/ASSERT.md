## Expected

1. `DownloadErr` is non-empty.
2. `DownloadAttempts` equals `MaxDownloadAttempts` (3).
3. `ResultSize` is zero or local file shorter than `FileSize`.

## Side Effects

No complete file on disk.

## Errors

- Empty `DownloadErr` (download should fail).
- `DownloadAttempts < MaxDownloadAttempts`.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.DownloadErr == "" {
		t.Fatal("expected download error, got success")
	}
	if !strings.Contains(strings.ToLower(resp.DownloadErr), "502") && !strings.Contains(strings.ToLower(resp.DownloadErr), "bad gateway") {
		t.Fatalf("DownloadErr = %q, want mention of 502/bad gateway", resp.DownloadErr)
	}
	if resp.DownloadAttempts != req.MaxDownloadAttempts {
		t.Fatalf("DownloadAttempts = %d, want %d", resp.DownloadAttempts, req.MaxDownloadAttempts)
	}
	if int64(len(resp.LocalFileContent)) == req.FileSize {
		t.Fatalf("local file complete (%d bytes) after exhaustion; want failure", len(resp.LocalFileContent))
	}
}
```