## Expected Output

Two-phase SSE apply stream: **CLASSIFYING** lists every entry (skip/update/create);
**APPLYING** repeats only non-skip actions; summary ends with `machine restore summary`.

```
CLASSIFYING:
skip (identical): .ai-critic/ai-models.json
update: .bashrc
...more skip lines...

APPLYING:
update: .bashrc

machine restore summary
  home:         ...
  skip (identical):  ...
  update:            1
  create:            0
  TOTAL: ... entries
```

## Expected

1. Exit code 0.
2. Combined output has `CLASSIFYING:` then `APPLYING:` (apply only).
3. CLASSIFYING includes `update: .bashrc` and at least one `skip (identical):` line.
4. CLASSIFYING does not include `skip (identical): .bashrc`.
5. APPLYING includes `update: .bashrc` only (no skip lines).
6. Combined output contains `machine restore summary` with `TOTAL:` counts.
7. `.bashrc` on server home equals seed content `export FAKE=1\n`.

## Side Effects

`.bashrc` reverted from mutated content to backup content.

## Errors

- Missing CLASSIFYING or APPLYING section.
- `.bashrc` still mutated or listed only as identical skip.
- Missing skip lines for unchanged paths in CLASSIFYING.
- Dry-run summary title instead of apply summary.

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
	assertRestoreStreamSections(t, combined, true)

	classify := restoreClassifyingSection(combined)
	assert.Output(t, classify, `---
version: 2
---
CLASSIFYING:
...0 lines omitted...
update: .bashrc
...9 lines omitted...`)
	if strings.Contains(classify, "skip (identical): .bashrc") {
		t.Fatalf(".bashrc should have been updated, not skipped in CLASSIFYING; section:\n%s", classify)
	}
	if !strings.Contains(classify, "skip (identical):") {
		t.Fatalf("expected skip lines for unchanged paths in CLASSIFYING; section:\n%s", classify)
	}

	applying := restoreApplyingSection(combined)
	assert.Output(t, applying, `---
version: 2
---
APPLYING:
update: .bashrc`)
	if strings.Contains(applying, "skip (identical):") {
		t.Fatalf("APPLYING must not repeat skip lines; section:\n%s", applying)
	}

	summary := restoreSummaryRest(combined)
	if summary == "" {
		t.Fatalf("missing machine restore summary; got:\n%s", combined)
	}
	assert.Output(t, summary, `---
version: 2
---
machine restore summary
...5 lines omitted...
  TOTAL: .+ entries`)

	got := readServerFile(t, resp.ServerHome, ".bashrc")
	want := "export FAKE=1\n"
	if got != want {
		t.Fatalf(".bashrc not restored: got %q want %q", got, want)
	}
}
```