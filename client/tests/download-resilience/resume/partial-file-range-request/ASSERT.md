## Expected

1. `DownloadErr` is empty.
2. At least one `Range` header equals `bytes=4096-`.
3. `DownloadAttempts` is 1 (single ranged GET for remaining bytes).
4. `LocalFileContent` matches `WantContent` (8192 bytes).

## Side Effects

Second half appended without truncating pre-filled prefix.

## Errors

- Missing or wrong `Range` header.
- Full re-download (`bytes=0-` only) without resume offset.
- Content mismatch.

```go
import (
	"bytes"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.DownloadErr != "" {
		t.Fatalf("DownloadFile failed: %s", resp.DownloadErr)
	}

	wantRange := "bytes=4096-"
	foundRange := false
	for _, h := range resp.RangeHeaders {
		if h == wantRange {
			foundRange = true
			break
		}
	}
	if !foundRange {
		t.Fatalf("RangeHeaders = %v, want containing %q", resp.RangeHeaders, wantRange)
	}
	if resp.DownloadAttempts != 1 {
		t.Fatalf("DownloadAttempts = %d, want 1 (single ranged GET)", resp.DownloadAttempts)
	}
	if !bytes.Equal(resp.LocalFileContent, resp.WantContent) {
		t.Fatalf("local content mismatch: got %d bytes, want %d", len(resp.LocalFileContent), len(resp.WantContent))
	}
	if !strings.HasPrefix(string(resp.LocalFileContent), string(resp.WantContent[:4096])) {
		t.Fatal("prefilled prefix was corrupted")
	}
}
```