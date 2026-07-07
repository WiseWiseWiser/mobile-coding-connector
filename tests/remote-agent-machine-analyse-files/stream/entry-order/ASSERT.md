## Expected Output

Top-level `> <entry>` headers appear in strict alphabetical order:
`.codex`, `aaa-first`, `mmm-mid`, `notes.txt`, `zzz-last`.

## Expected

1. Exit code 0.
2. Extracted entry header sequence is sorted ascending.
3. Sequence includes all five seeded entry names.

## Side Effects

None.

## Errors

- Out-of-order entry blocks.
- Missing seeded entries.

## Exit Code

0.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	names := extractEntryBlockOrder(t, resp.Combined)
	assertSortedEntryNames(t, names)

	want := []string{".codex", "aaa-first", "mmm-mid", "notes.txt", "zzz-last"}
	for _, entry := range want {
		found := false
		for _, n := range names {
			if n == entry {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing entry %q in header sequence %v", entry, names)
		}
	}
}
```