## Expected Output

Dry-run streams **CLASSIFYING** only (no APPLYING). Mutated `.bashrc` is `update:`;
unchanged paths are `skip (identical):`. Summary is `dry-run: machine restore plan`.

```
CLASSIFYING:
skip (identical): .ai-critic/ai-models.json
update: .bashrc
...more skip lines...

dry-run: machine restore plan
  home:         ...
  skip (identical):  ...
  update:            1
  create:            0
  TOTAL: ... entries
```

## Expected

1. Exit code 0.
2. Combined output has `CLASSIFYING:` and no `APPLYING:`.
3. CLASSIFYING contains `update:` referencing `.bashrc`.
4. CLASSIFYING contains at least one other `skip (identical):` line.
5. CLASSIFYING does not contain `skip (identical): .bashrc`.
6. Combined output contains `dry-run: machine restore plan` with `TOTAL:` counts.
7. Mutated on-disk `.bashrc` remains until apply (`mutated after backup`).

## Side Effects

None (dry-run).

## Errors

- `.bashrc` listed only as identical skip.
- Missing CLASSIFYING section or `update:` line.
- Unexpected APPLYING section.
- Missing restore plan summary.
- Server file reverted during dry-run.

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
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	combined := resp.Combined
	assertRestoreStreamSections(t, combined, false)

	classify := restoreClassifyingSection(combined)
	assert.Output(t, classify, `---
version: 2
---
CLASSIFYING:
...0 lines omitted...
update: .bashrc
...9 lines omitted...`)
	if strings.Contains(classify, "skip (identical): .bashrc") {
		t.Fatalf(".bashrc should not be identical after mutation; section:\n%s", classify)
	}
	if !strings.Contains(classify, "skip (identical):") {
		t.Fatalf("expected some identical skips in CLASSIFYING; section:\n%s", classify)
	}

	summary := restoreSummaryRest(combined)
	if summary == "" {
		t.Fatalf("missing dry-run restore plan summary; got:\n%s", combined)
	}
	assert.Output(t, summary, `---
version: 2
---
dry-run: machine restore plan
...5 lines omitted...
  TOTAL: .+ entries`)

	got := readServerFile(t, resp.ServerHome, ".bashrc")
	want := "mutated after backup\n"
	if got != want {
		t.Fatalf(".bashrc should remain mutated during dry-run: got %q want %q", got, want)
	}
}
```