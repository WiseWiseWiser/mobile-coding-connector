## Expected Output

Stream phase prints `update:` for mutated `.bashrc` and `skip (identical):` for
unchanged paths. Summary phase prints `dry-run: machine restore plan` with counts.

## Expected

1. Exit code 0.
2. Combined output contains `update:` referencing `.bashrc` (stream phase).
3. Combined output contains `dry-run: machine restore plan` with skip/update/create counts.
4. Mutated on-disk `.bashrc` remains until apply (`mutated after backup`).
5. At least one other path still reports `skip (identical):` (e.g. `.ai-critic/...`).

## Side Effects

None (dry-run).

## Errors

- `.bashrc` listed only as identical skip.
- Missing `update:` stream line or restore summary.
- Server file reverted during dry-run.

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

	combined := resp.Combined
	if !strings.Contains(combined, "update:") {
		t.Fatalf("expected update: stream line; got:\n%s", combined)
	}
	if !strings.Contains(strings.ToLower(combined), ".bashrc") {
		t.Fatalf("output missing .bashrc plan; got:\n%s", combined)
	}
	if !strings.Contains(combined, "dry-run: machine restore plan") {
		t.Fatalf("missing restore plan summary; got:\n%s", combined)
	}
	if !strings.Contains(combined, "TOTAL:") {
		t.Fatalf("restore summary missing TOTAL entry count; got:\n%s", combined)
	}

	if strings.Contains(combined, "skip (identical): .bashrc") {
		t.Fatalf(".bashrc should not be identical after mutation; got:\n%s", combined)
	}
	if !strings.Contains(combined, "skip (identical):") {
		t.Fatalf("expected some identical skips; got:\n%s", combined)
	}

	got := readServerFile(t, resp.ServerHome, ".bashrc")
	want := "mutated after backup\n"
	if got != want {
		t.Fatalf(".bashrc should remain mutated during dry-run: got %q want %q", got, want)
	}
}
```