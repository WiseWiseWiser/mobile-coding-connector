## Expected

1. `DownloadErr` is empty — download succeeds.
2. `DownloadAttempts` is 3 (two failures + one success).
3. `ResultSize` equals `FileSize`.
4. `LocalFileContent` matches `WantContent`.

## Side Effects

Mock server serves full byte stream matching source.

## Errors

- Non-empty `DownloadErr`.
- `DownloadAttempts < 3`.
- Content mismatch.

```go
import (
	"bytes"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.DownloadErr != "" {
		t.Fatalf("DownloadFile failed: %s", resp.DownloadErr)
	}
	if resp.DownloadAttempts != 3 {
		t.Fatalf("DownloadAttempts = %d, want 3 (2 retries + 1 success)", resp.DownloadAttempts)
	}
	if resp.ResultSize != req.FileSize {
		t.Fatalf("ResultSize = %d, want %d", resp.ResultSize, req.FileSize)
	}
	if !bytes.Equal(resp.LocalFileContent, resp.WantContent) {
		t.Fatalf("local content mismatch: got %d bytes, want %d", len(resp.LocalFileContent), len(resp.WantContent))
	}
}
```