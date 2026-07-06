## Expected Output

```
skip (identical): .ai-critic/ai-models.json
```

`.bashrc` restored to archive bytes without an identical skip line.

## Expected

1. Exit code 0.
2. `.bashrc` on server home equals seed content `export FAKE=1\n`.
3. Combined output includes `skip (identical):` for at least one unchanged path.
4. Combined output does not include `skip (identical): .bashrc`.

## Side Effects

`.bashrc` reverted from mutated content to backup content.

## Errors

- `.bashrc` still mutated.
- Missing skip lines for unchanged paths.

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

	got := readServerFile(t, resp.ServerHome, ".bashrc")
	want := "export FAKE=1\n"
	if got != want {
		t.Fatalf(".bashrc not restored: got %q want %q", got, want)
	}

	if strings.Contains(resp.Combined, "skip (identical): .bashrc") {
		t.Fatalf(".bashrc should have been updated, not skipped; output:\n%s", resp.Combined)
	}
	if !strings.Contains(resp.Combined, "skip (identical):") {
		t.Fatalf("expected skip lines for unchanged paths; got:\n%s", resp.Combined)
	}
}
```