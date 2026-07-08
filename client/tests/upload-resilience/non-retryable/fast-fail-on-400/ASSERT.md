## Expected

1. `UploadErr` is non-empty.
2. `ChunkAttempts[1]` is 1 (no retry on 400).
3. `ChunkAttempts[0]` is 1.
4. `CompleteCalled` is false.

## Side Effects

None beyond partial chunk 0 stored on mock server.

## Errors

- `ChunkAttempts[1] > 1`.
- Empty `UploadErr`.

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
	if !strings.Contains(resp.UploadErr, "chunk 1") {
		t.Fatalf("UploadErr = %q, want mention of chunk 1", resp.UploadErr)
	}
	if resp.ChunkAttempts[1] != 1 {
		t.Fatalf("chunk 1 attempts = %d, want 1 (no retry on 400)", resp.ChunkAttempts[1])
	}
	if resp.ChunkAttempts[0] != 1 {
		t.Fatalf("chunk 0 attempts = %d, want 1", resp.ChunkAttempts[0])
	}
	if resp.CompleteCalled {
		t.Fatal("complete should not be called after 400")
	}
}
```