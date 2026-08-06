## Expected

1. Harness `err` is nil.
2. `EnsureErr` / `EnsureErr2` empty; private key exists on disk.
3. `ServeErr` empty; `SessionAfterStart` Alive with LocalPort > 0.
4. `RunnerErr` empty (must not contain `signer not configured`).
5. `Stdout` contains `EchoNeedle` (`signer-ok`).
6. `ReloadedSame` true (same identity for authorize + run).
7. `AdhocPort` and `RelayLocalPort` > 0.

## Side Effects

- `{ConfigDir}/id_ed25519` created; session file under Root during serve; cleared on cancel.

## Errors

None expected once Ensure + existing compose work. Classic RED while Ensure is missing.

```go
import (
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
		t.Fatalf("EnsureClientKeyPair (create/load): %s", resp.EnsureErr)
	}
	if resp.EnsureErr2 != "" {
		t.Fatalf("EnsureClientKeyPair (reload for runner): %s", resp.EnsureErr2)
	}
	if !resp.PrivateExists {
		t.Fatal("id_ed25519 must exist on disk for load-from-disk path")
	}
	if !resp.ReloadedSame {
		t.Fatalf("authorize Public and runner identity must match: %q vs %q", resp.PublicMaterial, resp.PublicMaterial2)
	}
	if resp.ServeErr != "" {
		t.Fatalf("ServeService error: %s", resp.ServeErr)
	}
	if resp.SessionAfterStart == nil || !resp.SessionAfterStart.Alive {
		t.Fatal("SessionAfterStart must be Alive after Serve Start")
	}
	if resp.SessionAfterStart.LocalPort <= 0 {
		t.Fatalf("SessionAfterStart.LocalPort: got %d", resp.SessionAfterStart.LocalPort)
	}
	if resp.RelayLocalPort <= 0 {
		t.Fatalf("RelayLocalPort: got %d want > 0", resp.RelayLocalPort)
	}
	if resp.AdhocPort <= 0 {
		t.Fatalf("AdhocPort: got %d want > 0", resp.AdhocPort)
	}
	if resp.RunnerErr != "" {
		t.Fatalf("CryptoSSHRunner with disk Signer: %s (stderr=%q)", resp.RunnerErr, resp.Stderr)
	}
	if strings.Contains(strings.ToLower(resp.RunnerErr), "signer") {
		// defensive: empty RunnerErr already checked; keep message stable if logic changes
		t.Fatalf("unexpected signer error after Ensure wire: %s", resp.RunnerErr)
	}
	needle := req.EchoNeedle
	if needle == "" {
		needle = "signer-ok"
	}
	if !strings.Contains(resp.Stdout, needle) {
		t.Fatalf("Stdout must contain %q; got %q (stderr=%q)", needle, resp.Stdout, resp.Stderr)
	}
}
```
