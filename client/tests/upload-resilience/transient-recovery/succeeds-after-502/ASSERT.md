## Expected

1. `UploadErr` is empty — upload succeeds.
2. `CompleteCalled` is true.
3. `InitCount` is 1 (single session, no restart from chunk 0).
4. `ChunkAttempts[2]` is 3 (two failures + one success).
5. `TotalChunkPosts` is 7 (5 chunks + 2 extra retries on chunk 2).
6. `ResultSize` equals `TotalBytes`.

## Side Effects

Mock server assembles full byte stream matching source file.

## Errors

- Non-empty `UploadErr`.
- `ChunkAttempts[2] < 3`.
- `InitCount != 1`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
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
		t.Fatalf("InitCount = %d, want 1 (no session restart)", resp.InitCount)
	}
	attempts := resp.ChunkAttempts[2]
	if attempts != 3 {
		t.Fatalf("chunk 2 attempts = %d, want 3 (2 retries + 1 success)", attempts)
	}
	wantPosts := 7
	if resp.TotalChunkPosts != wantPosts {
		t.Fatalf("TotalChunkPosts = %d, want %d", resp.TotalChunkPosts, wantPosts)
	}
	if resp.ResultSize != req.TotalBytes {
		t.Fatalf("ResultSize = %d, want %d", resp.ResultSize, req.TotalBytes)
	}
}
```