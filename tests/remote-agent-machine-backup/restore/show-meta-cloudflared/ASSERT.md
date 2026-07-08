## Expected Output

Stdout prints `=== .backup/cloudflared-config.json ===` with archive content including
`captured_at`, `"running": true`, and redacted config. Does not print `config.json`.

## Expected

1. Exit code 0.
2. Combined output contains `=== .backup/cloudflared-config.json ===` and `captured_at`.
3. Combined output contains `"running": true`.
4. Combined output does not contain raw tunnel/credentials secrets from mock config.
5. Combined output does not contain `=== .backup/config.json ===`.

## Side Effects

None (read-only archive inspection).

## Errors

- Missing cloudflared meta section or leaked secrets.

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
		"=== .backup/cloudflared-config.json ===",
		"captured_at",
		`"running": true`,
	)
	combinedHasNone(t, resp.Combined,
		"=== .backup/config.json ===",
		cloudflaredFixtureTunnelID,
		cloudflaredFixtureCredFile,
	)

	raw := tarXZExtractFile(t, resp.BackupPath, ".backup/cloudflared-config.json")
	snap := parseCloudflaredConfigJSON(t, raw)
	assertCloudflaredConfigBasics(t, snap)
	assertCloudflaredConfigRedacted(t, snap)
	if !strings.Contains(resp.Combined, snap.CapturedAt) {
		t.Fatalf("stdout missing captured_at %q; got:\n%s", snap.CapturedAt, resp.Combined)
	}
}
```