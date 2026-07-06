## Expected Output

Stdout is indented built-in exclusion config JSON.

## Expected

1. Exit code 0.
2. Stdout parses as exclusion config with `version` `1.0`.
3. `exclude_paths` includes `.cache` with a non-empty `reason`.

## Side Effects

None (no archive read, no server restore).

## Errors

- Invalid JSON or missing fields.

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

	cfg := parseExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.Stdout)))
	if cfg.Version != "1.0" {
		t.Fatalf("version = %q, want 1.0", cfg.Version)
	}
	foundCache := false
	for _, e := range cfg.ExcludePaths {
		if e.Path == ".cache" {
			foundCache = true
			if strings.TrimSpace(e.Reason) == "" {
				t.Fatal(".cache exclusion missing reason")
			}
		}
	}
	if !foundCache {
		t.Fatalf("exclude_paths missing .cache: %+v", cfg.ExcludePaths)
	}
}
```