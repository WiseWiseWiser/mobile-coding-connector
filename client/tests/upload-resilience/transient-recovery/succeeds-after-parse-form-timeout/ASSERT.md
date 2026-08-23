## Expected

1. `UploadErr` is empty — upload succeeds after retries.
2. `CompleteCalled` is true.
3. `InitCount` is 1.
4. `ChunkAttempts[2]` is 3 (two timeout-as-400 failures + one success).
5. `TotalChunkPosts` is 7.
6. `ResultSize` equals `TotalBytes`.

## Side Effects

Same as other transient-recovery leaves.

## Errors

- Non-empty `UploadErr` (would mean 400 was treated as fatal).
- `ChunkAttempts[2] != 3`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.UploadErr != "" {
		t.Fatalf("UploadFile failed: %s", resp.UploadErr)
	}
	if !resp.CompleteCalled {
		t.Fatal("complete endpoint was not called")
	}
	if resp.InitCount != 1 {
		t.Fatalf("InitCount = %d, want 1", resp.InitCount)
	}
	if resp.ChunkAttempts[2] != 3 {
		t.Fatalf("chunk 2 attempts = %d, want 3", resp.ChunkAttempts[2])
	}
	if resp.TotalChunkPosts != 7 {
		t.Fatalf("TotalChunkPosts = %d, want 7", resp.TotalChunkPosts)
	}
	if resp.ResultSize != req.TotalBytes {
		t.Fatalf("ResultSize = %d, want %d", resp.ResultSize, req.TotalBytes)
	}
}
```
