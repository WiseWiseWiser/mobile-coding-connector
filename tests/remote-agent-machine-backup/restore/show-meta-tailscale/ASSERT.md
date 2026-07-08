## Expected Output

Stdout prints `=== .backup/tailscale-config.json ===` with archive content including
`captured_at`, `"running": true`, and redacted prefs. Does not print `config.json`.

## Expected

1. Exit code 0.
2. Combined output contains `=== .backup/tailscale-config.json ===` and `captured_at`.
3. Combined output contains `"running": true`.
4. Combined output does not contain raw fake private keys from mock prefs.
5. Combined output does not contain `=== .backup/config.json ===`.

## Side Effects

None (read-only archive inspection).

## Errors

- Missing tailscale meta section or leaked private keys.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if resp.BackupPath == "" {
		t.Fatal("prereq BackupPath empty")
	}

	combinedHasAll(t, resp.Combined,
		"=== .backup/tailscale-config.json ===",
		"captured_at",
		`"running": true`,
	)
	combinedHasNone(t, resp.Combined,
		"=== .backup/config.json ===",
		"nodekey:fake-private-should-redact",
		"nodekey:fake-old-should-redact",
		"nlkey:fake-lock-should-redact",
		"nodekey:fake-nested-should-redact",
	)

	raw := tarXZExtractFile(t, resp.BackupPath, ".backup/tailscale-config.json")
	snap := parseTailscaleConfigJSON(t, raw)
	assertTailscaleConfigBasics(t, snap)
	assertTailscalePrefsRedacted(t, snap.Prefs)
	if !strings.Contains(resp.Combined, snap.CapturedAt) {
		t.Fatalf("stdout missing captured_at %q; got:\n%s", snap.CapturedAt, resp.Combined)
	}
}
```