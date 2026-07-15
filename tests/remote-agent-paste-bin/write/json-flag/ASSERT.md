## Expected

1. Exit code 0.
2. Stdout is valid JSON with `content` and `updated_at` from PUT response.
3. JSON `content` equals written payload `hi`.
4. Stdout has no ANSI; no preview or echo beyond JSON.
5. Stderr silent (no `saved N bytes` on `--json` write).

## Side Effects

Scratch API updated to `hi`.

## Errors

- `saved N bytes` preview on stderr when `--json` is set.
- Raw `hi` echoed outside JSON on stdout.
- ANSI in JSON stdout.

## Exit Code

0.

```go
import (
	"encoding/json"
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
	if resp.Stderr != "" {
		t.Fatalf("--json write must not print stderr summary; got:\n%s", resp.Stderr)
	}
	if strings.Contains(resp.Stdout, "\x1b[") {
		t.Fatalf("--json stdout must not contain ANSI; got:\n%s", resp.Stdout)
	}

	var payload ScratchEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, resp.Stdout)
	}
	if payload.Content != smallEchoPayload {
		t.Fatalf("json content = %q, want %q", payload.Content, smallEchoPayload)
	}
	if payload.UpdatedAt == "" {
		t.Fatalf("json updated_at must be populated; stdout:\n%s", resp.Stdout)
	}
	if resp.Stdout != strings.TrimSpace(resp.Stdout)+"\n" && !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("json stdout should end with newline when applicable; got %q", resp.Stdout)
	}
	assertScratchContentExact(t, resp.ScratchAfter.Content, smallEchoPayload)
}
```