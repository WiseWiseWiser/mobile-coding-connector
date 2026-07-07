## Expected Output

`notes.txt` block shows `lines   2`; `binary.dat` block shows `lines   (binary)`.

## Expected

1. Exit code 0.
2. `notes.txt` entry block contains `size` and `lines   2` (whitespace flexible).
3. `binary.dat` entry block contains `size` and `lines   (binary)`.

## Side Effects

None.

## Errors

- Text file missing line count.
- Binary file not marked `(binary)`.

## Exit Code

0.

```go
import (
	"regexp"
	"testing"
)

var textLinesRE = regexp.MustCompile(`(?m)^\s*lines\s+2\s*$`)
var binaryLinesRE = regexp.MustCompile(`(?m)^\s*lines\s+\(binary\)\s*$`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	notesBlock := extractEntryBlock(t, resp.Combined, "notes.txt")
	binaryBlock := extractEntryBlock(t, resp.Combined, "binary.dat")

	combinedHasAll(t, notesBlock, "size")
	combinedHasAll(t, binaryBlock, "size")

	if !textLinesRE.MatchString(notesBlock) {
		t.Fatalf("notes.txt block missing lines 2:\n%s", notesBlock)
	}
	if !binaryLinesRE.MatchString(binaryBlock) {
		t.Fatalf("binary.dat block missing lines (binary):\n%s", binaryBlock)
	}
}
```