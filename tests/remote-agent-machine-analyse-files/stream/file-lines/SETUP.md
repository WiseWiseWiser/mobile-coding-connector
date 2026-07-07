# Scenario

**Feature**: top-level files report size and line counts

```
# text file -> lines N; binary file -> lines (binary)
notes.txt + binary.dat at serverHome root
```

## Preconditions

`SeedProfile=file-lines`: `testdata/notes.txt` (2 lines) and `testdata/binary.dat` (NUL bytes).

## Steps

1. Set `SeedProfile` to `file-lines`.

## Context

REQUIREMENT leaf `stream/file-lines`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedProfile = "file-lines"
	req.Args = []string{"machine", "analyse-files"}
	return nil
}
```