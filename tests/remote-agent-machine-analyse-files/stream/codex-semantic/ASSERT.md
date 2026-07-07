## Expected Output

`.codex` block lists immediate children (`> sessions`, `> skills`, …) before
semantic lines (`sessions  2 rollouts`, `skills  1 skill`). Summary includes
`codex sessions:` and `codex skills:` when `.codex` exists.

## Expected

1. Exit code 0.
2. `.codex` entry block exists.
3. Child line `> sessions` appears before semantic `sessions` count line.
4. Child line `> skills` appears before semantic `skills` count line.
5. Semantic lines report `2 rollouts` and `1 skill` (or equivalent counts).
6. Summary contains `codex sessions:` and `codex skills:`.

## Side Effects

None.

## Errors

- Semantic lines before children.
- Wrong rollout/skill counts.
- Missing codex summary lines.

## Exit Code

0.

```go
import (
	"regexp"
	"testing"
)

var codexCountLine = regexp.MustCompile(`(?m)^\s*sessions\s+\d+\s+rollouts\b`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	combinedHasAll(t, resp.Combined,
		"> .codex",
		"analyse-files summary",
		"codex sessions:",
		"codex skills:",
	)

	block := extractEntryBlock(t, resp.Combined, ".codex")
	assertChildBeforeSemantic(t, block, "> sessions", "sessions")
	assertChildBeforeSemantic(t, block, "> skills", "skills")

	if !codexCountLine.MatchString(block) {
		t.Fatalf(".codex block missing sessions rollout count line:\n%s", block)
	}
	combinedHasAll(t, block, "1 skill")
}
```