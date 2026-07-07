## Expected Output

Stdout effective JSON includes `.knowledge-index` with reason `user excluded` and trailing newline.

## Expected

1. Exit code 0.
2. Stdout parses as effective exclusion config with trailing newline.
3. Effective `exclude_paths` includes `.knowledge-index` with reason `user excluded`.
4. No restore writes and no archive read.

## Side Effects

None (preview only).

## Errors

- CLI `--exclude` ignored on restore show-config without archive.
- Reason is not `user excluded`.
- Server home files mutated.

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

	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("stdout missing trailing newline; got %q", resp.Stdout)
	}
	cfg := parseEffectiveExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.Stdout)))
	assertEffectiveExcludeReason(t, cfg, ".knowledge-index", "user excluded")
}
```