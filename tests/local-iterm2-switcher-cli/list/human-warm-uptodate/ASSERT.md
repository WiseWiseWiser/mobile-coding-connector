## Expected

1. Exit 0.
2. No HTTP leak (`HitHTTP` false — no daemon resolve / `/api` noise).
3. `CaptureCalls == 0` (same IDs: Layout only).
4. `LayoutCalls >= 1` (incremental layout-diff ran).
5. Human stdout includes `Desktop 1` and `grok review`.
6. Mentions incremental **or** up to date.
7. Stdout ends with `\n`.

## Errors

- Still uses HTTP client; deep Capture on warm same-layout; unknown command.

```go
import (
	"regexp"
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit=%d out=%q err=%q", resp.ExitCode, resp.Combined, resp.ErrMsg)
	}
	if resp.HitHTTP {
		t.Fatalf("list must not talk to daemon/HTTP; out=%q err=%q", resp.Combined, resp.ErrMsg)
	}
	if resp.CaptureCalls != 0 {
		t.Fatalf("CaptureCalls=%d want 0 (warm same IDs: Layout only)", resp.CaptureCalls)
	}
	if resp.LayoutCalls < 1 {
		t.Fatalf("LayoutCalls=%d want ≥ 1 (incremental layout-diff)", resp.LayoutCalls)
	}
	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatal("stdout must end with newline")
	}
	if !regexp.MustCompile(`Desktop 1`).MatchString(resp.Stdout) {
		t.Fatalf("human list missing FormatDesktopHeader; stdout=%q", resp.Stdout)
	}
	if !regexp.MustCompile(`grok review`).MatchString(resp.Stdout) {
		t.Fatalf("human list missing FormatSessionPrimary; stdout=%q", resp.Stdout)
	}
	if !regexp.MustCompile(`(?i)incremental|up to date`).MatchString(resp.Stdout + "\n" + resp.Stderr) {
		t.Fatalf("warm list must mention incremental / up to date; out=%q", resp.Combined)
	}
}
```
