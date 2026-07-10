# Scenario

**Feature**: JoinBackupProgressBatch joins lines with newline and trailing newline

```
# sealed pure API (macosapp/menubar)
JoinBackupProgressBatch(["a","b"]) -> "a\nb\n"
JoinBackupProgressBatch([])        -> ""
JoinBackupProgressBatch(["solo"])  -> "solo\n"

# design-phase contract: source must define func + policy body
# (empty guard, strings.Join with "\n", trailing newline)
```

## Preconditions

`func JoinBackupProgressBatch` in `macosapp/menubar` with empty→`""` and join+trailing `\n`.

## Steps

1. Op=helper_join_batch; BatchLines=["a","b"] (documents intended pure call).

## Context

REQUIREMENT join policy for one textStorage append per flush.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "helper_join_batch"
	req.BatchLines = []string{"a", "b"}
	return nil
}
```
