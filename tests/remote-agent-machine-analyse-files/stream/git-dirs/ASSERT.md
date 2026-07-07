## Expected Output

`with-git` entry block ends with `git-dirs  1` (or spaced variant). `plain-dir`
block does not contain a `git-dirs` line.

## Expected

1. Exit code 0.
2. `with-git` block contains `git-dirs` with count `1`.
3. `plain-dir` block omits `git-dirs` line entirely.

## Side Effects

None.

## Errors

- Git entry missing `git-dirs`.
- Non-git entry shows `git-dirs`.

## Exit Code

0.

```go
import (
	"regexp"
	"strings"
	"testing"
)

var gitDirsOneRE = regexp.MustCompile(`(?m)^\s*git-dirs\s+1\s*$`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	withGit := extractEntryBlock(t, resp.Combined, "with-git")
	plainDir := extractEntryBlock(t, resp.Combined, "plain-dir")

	if !gitDirsOneRE.MatchString(withGit) {
		t.Fatalf("with-git block missing git-dirs 1:\n%s", withGit)
	}
	if strings.Contains(plainDir, "git-dirs") {
		t.Fatalf("plain-dir block should omit git-dirs; got:\n%s", plainDir)
	}
}
```