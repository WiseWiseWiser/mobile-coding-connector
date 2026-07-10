## Expected Output

Error text mentions `--cron-utc` so the user can pass an explicit UTC expression.

## Expected

1. CLI exit code **non-zero**.
2. Combined stdout/stderr contains `--cron-utc` (case-sensitive flag form preferred).
3. Task `unsafe-cron` is **not** present in list (create aborted).

## Side Effects

- No persisted definition for the failed add.

## Errors

- Exit 0 on unsafe convert.
- Missing guidance to use `--cron-utc`.

## Exit Code

Non-zero.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run harness error: %v", err)
	}
	if resp.ExitCode == 0 && resp.ActionError == "" {
		t.Fatalf("want non-zero exit for unsafe --cron, got 0\n%s", resp.Combined)
	}
	combined := resp.Combined
	if combined == "" {
		combined = resp.Stdout + "\n" + resp.Stderr
	}
	if !strings.Contains(combined, "--cron-utc") {
		t.Fatalf("error message must mention --cron-utc:\n%s", combined)
	}
	assert.Output(t, combined, `<contains>
--cron-utc
</contains>`)

	if _, ok := findTaskByName(resp.Tasks, "unsafe-cron"); ok {
		t.Fatal("unsafe-cron was created despite convert failure")
	}
}
```
