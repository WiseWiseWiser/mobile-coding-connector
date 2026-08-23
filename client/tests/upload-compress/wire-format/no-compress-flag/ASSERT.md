## Expected

1. Upload succeeds.
2. Init sent `compressed=false`.
3. `WireTotalSize` equals original length.
4. `ResultSize` equals original length.

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
	if resp.InitCompressed == nil || *resp.InitCompressed {
		t.Fatal("expected init compressed=false")
	}
	orig := int64(len(req.Content))
	if resp.WireTotalSize != orig {
		t.Fatalf("WireTotalSize=%d want %d", resp.WireTotalSize, orig)
	}
	if resp.ResultSize != orig {
		t.Fatalf("ResultSize=%d want %d", resp.ResultSize, orig)
	}
}
```
