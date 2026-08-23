## Expected

1. Upload succeeds (`UploadErr` empty, `CompleteCalled`).
2. Init sent `compressed=true`.
3. `WireTotalSize` < original content length.
4. `ResultSize` equals original content length.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.UploadErr != "" {
		t.Fatalf("upload failed: %s", resp.UploadErr)
	}
	if !resp.CompleteCalled {
		t.Fatal("complete not called")
	}
	if resp.InitCompressed == nil || !*resp.InitCompressed {
		t.Fatal("expected init compressed=true")
	}
	orig := int64(len(req.Content))
	if resp.WireTotalSize >= orig {
		t.Fatalf("WireTotalSize=%d not smaller than original %d", resp.WireTotalSize, orig)
	}
	if resp.ResultSize != orig {
		t.Fatalf("ResultSize=%d want %d", resp.ResultSize, orig)
	}
}
```
