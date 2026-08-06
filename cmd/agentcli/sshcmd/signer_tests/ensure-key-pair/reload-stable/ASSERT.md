## Expected

1. Harness `err` is nil.
2. `EnsureErr` and `EnsureErr2` empty.
3. `SignerNonNil` and `PublicNonNil` true.
4. `PublicMaterial` and `PublicMaterial2` non-empty and equal.
5. `ReloadedSame` is true.
6. Private file still exists under ConfigDir.

## Side Effects

- Single `id_ed25519` on disk after both calls (no second identity).

## Errors

None expected once Ensure is implemented.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.EnsureErr != "" {
		t.Fatalf("first EnsureClientKeyPair error: %s", resp.EnsureErr)
	}
	if resp.EnsureErr2 != "" {
		t.Fatalf("second EnsureClientKeyPair error: %s", resp.EnsureErr2)
	}
	if !resp.SignerNonNil || !resp.PublicNonNil {
		t.Fatal("both Ensure calls must return non-nil Signer and Public")
	}
	if resp.PublicMaterial == "" || resp.PublicMaterial2 == "" {
		t.Fatalf("public materials empty: first=%q second=%q", resp.PublicMaterial, resp.PublicMaterial2)
	}
	if resp.PublicMaterial != resp.PublicMaterial2 {
		t.Fatalf("public key not stable:\n first=%q\nsecond=%q", resp.PublicMaterial, resp.PublicMaterial2)
	}
	if !resp.ReloadedSame {
		t.Fatal("ReloadedSame must be true when public materials match")
	}
	if !resp.PrivateExists {
		t.Fatal("id_ed25519 must remain on disk after reload")
	}
}
```
