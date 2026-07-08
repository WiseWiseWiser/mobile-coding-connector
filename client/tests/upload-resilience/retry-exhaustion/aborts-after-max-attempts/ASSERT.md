## Expected

1. `UploadErr` is non-empty.
2. `ChunkAttempts[0]` equals `MaxChunkAttempts` (3).
3. `CompleteCalled` is false.
4. `InitCount` is 1.

## Side Effects

No assembled file on server.

## Errors

- Empty `UploadErr` (upload should fail).
- `CompleteCalled` true.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.UploadErr == "" {
		t.Fatal("expected upload error, got success")
	}
	if !strings.Contains(resp.UploadErr, "chunk 0") {
		t.Fatalf("UploadErr = %q, want mention of chunk 0", resp.UploadErr)
	}
	if resp.ChunkAttempts[0] != req.MaxChunkAttempts {
		t.Fatalf("chunk 0 attempts = %d, want %d", resp.ChunkAttempts[0], req.MaxChunkAttempts)
	}
	if resp.CompleteCalled {
		t.Fatal("complete should not be called after exhaustion")
	}
	if resp.InitCount != 1 {
		t.Fatalf("InitCount = %d, want 1", resp.InitCount)
	}
}
```