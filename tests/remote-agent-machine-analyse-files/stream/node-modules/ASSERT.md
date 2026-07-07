## Expected Output

`nm-entry` block lists immediate child `> node_modules` and aggregate
`node_modules  2 dirs` (two distinct `node_modules` directories under the entry).

## Expected

1. Exit code 0.
2. `nm-entry` block contains child line `> node_modules`.
3. `nm-entry` block contains aggregate `node_modules` with `2 dirs`.
4. Summary contains global `node_modules:` rollup with count >= 2.

## Side Effects

None.

## Errors

- Missing child `node_modules` line.
- Missing or wrong recursive dir count.

## Exit Code

0.

```go
import (
	"regexp"
	"testing"
)

var nodeModulesDirsRE = regexp.MustCompile(`(?m)^\s*node_modules\s+2\s+dirs\s*$`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	block := extractEntryBlock(t, resp.Combined, "nm-entry")
	combinedHasAll(t, block, "> node_modules")

	if !nodeModulesDirsRE.MatchString(block) {
		t.Fatalf("nm-entry block missing node_modules 2 dirs aggregate:\n%s", block)
	}

	combinedHasAll(t, resp.Combined, "node_modules:")
}
```