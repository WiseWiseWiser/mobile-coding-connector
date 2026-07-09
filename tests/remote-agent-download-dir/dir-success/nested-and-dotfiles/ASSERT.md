## Expected Output

```
Downloading uploads/dot-mirror -> ./local-dot/ (3 items, __SIZE__)
...progress lines omitted...
  [__IDX__/3] created emptydir/ — __PCT__% overall
...progress lines omitted...
Download complete: ./local-dot/ (2 files, __SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout banner reports `3 items` (2 regular files + 1 empty subdir); summary reports `2 files` (`.hidden`, `sub/.keep`).
3. Stdout contains a `created emptydir/` line with a `[N/M]` item index (e.g. `[2/3] created emptydir/ — X% overall`).
4. Local paths `.hidden`, `sub/.keep`, and empty directory `emptydir/` exist.

## Side Effects

- Dotfiles and empty subdirectory mirrored under `local-dot/`.

## Errors

- Dotfiles skipped.
- `emptydir/` not created locally.

## Exit Code

0.

```go
import (
	"regexp"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var reCreatedEmptyDir = regexp.MustCompile(`\[[0-9]+/3\] created emptydir/ — [0-9]+% overall`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
	combinedHasAll(t, resp.Combined, "3 items", "2 files", "local-dot", "created emptydir/", "overall")
	if !reCreatedEmptyDir.MatchString(resp.Stdout) {
		t.Fatalf("stdout missing indexed created emptydir/ line;\nhave:\n%s", resp.Stdout)
	}

	assertLocalFileContent(t, resp.LocalDir, ".hidden", "dotfile\n")
	assertLocalFileContent(t, resp.LocalDir, "sub/.keep", "")
	assertLocalIsDir(t, resp.LocalDir, "emptydir")
	assertLocalDirEmpty(t, resp.LocalDir, "emptydir")

	assert.Output(t, resp.Stdout, `---
version: 2
__SIZE__: type=string
__IDX__: type=number
__PCT__: type=number
---
Downloading uploads/dot-mirror -> ./local-dot/ (3 items, __SIZE__)
...2 lines omitted...
  [__IDX__/3] created emptydir/ — __PCT__% overall
...2 lines omitted...
Download complete: ./local-dot/ (2 files, __SIZE__)
`)
}
```