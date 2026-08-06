## Expected

1. Harness `err` is nil.
2. `EnsureErr` is empty.
3. `PrivateExists` is true; `PrivateKeyPath` ends with `id_ed25519` under ConfigDir.
4. `PrivateMode` is `0o600` (owner read/write only).
5. `SignerNonNil` and `PublicNonNil` are true.
6. `PublicMaterial` is non-empty (OpenSSH authorized_keys line).

## Side Effects

- Creates `{ConfigDir}/id_ed25519` (and may create ConfigDir / optional `.pub`).

## Errors

None expected on happy path. Classic RED until `EnsureClientKeyPair` exists.

```go
import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.EnsureErr != "" {
		t.Fatalf("EnsureClientKeyPair error: %s", resp.EnsureErr)
	}
	wantPath := filepath.Join(req.ConfigDir, "id_ed25519")
	if resp.PrivateKeyPath != wantPath {
		t.Fatalf("PrivateKeyPath: got %q want %q", resp.PrivateKeyPath, wantPath)
	}
	if !resp.PrivateExists {
		t.Fatal("id_ed25519 must exist after Ensure on missing key")
	}
	if resp.PrivateMode != 0o600 {
		t.Fatalf("id_ed25519 mode: got %#o want 0600", resp.PrivateMode)
	}
	if !resp.SignerNonNil {
		t.Fatal("ClientKeyPair.Signer must be non-nil")
	}
	if !resp.PublicNonNil {
		t.Fatal("ClientKeyPair.Public must be non-nil")
	}
	if strings.TrimSpace(resp.PublicMaterial) == "" {
		t.Fatal("PublicMaterial (authorized_keys line) must be non-empty")
	}
}
```
