# Scenario

**Feature**: resume sends Range header starting at prefill offset

```
# Range: bytes=4096- -> append second half
prefill 4096 B -> DownloadFile -> Range GET -> 8192 B total
```

## Preconditions

Inherited from `resume/SETUP.md`.

## Steps

No additional setup.

## Context

Leaf asserts Range header and appended content.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.LocalPrefillBytes != 4096 || req.FileSize != 8192 {
		t.Fatalf("parent setup: LocalPrefillBytes=%d FileSize=%d, want 4096/8192", req.LocalPrefillBytes, req.FileSize)
	}
	return nil
}
```