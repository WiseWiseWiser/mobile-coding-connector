## Expected Output

CLI stdout contains the stopped-service enable prompt mentioning daemon check.

## Expected

1. `Response.ExitCode` is 0.
2. `Response.Stdout` mentions deferred start (contains `daemon` and `next`, case-insensitive).
3. On-disk `services.json` has `enabled: true` for the target.

## Side Effects

- CLI prints API `message` to stdout.

## Errors

- Non-zero CLI exit code.
- Missing daemon prompt in stdout.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		combined := ""
		if resp != nil {
			combined = resp.Combined
		}
		t.Fatalf("Run error: %v\ncombined:\n%s", err, combined)
	}
	if resp.ActionError != "" {
		t.Fatalf("CLI run failed: %s", resp.ActionError)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assert.Output(t, strings.ToLower(resp.Stdout), `<contains>
daemon
next
</contains>`)

	enabled, present := enabledFieldOnDisk(resp.ServicesOnDisk, req.TargetID)
	if !present || enabled == nil || !*enabled {
		t.Fatalf("services.json enabled = %v present=%v, want true", enabled, present)
	}
}
```