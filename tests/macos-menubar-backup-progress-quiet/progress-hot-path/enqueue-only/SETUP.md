# Scenario

**Feature**: progress callback enqueues via session.append; session batches flush

```
downloadBackupArchive { event in
  line = Format…
  progressSession?.append(line)   # not textView.string += here
}
# session.append must buffer/flush, not immediate string += UI thrash
```

## Preconditions

1. onProgress / download path calls `progressSession?.append` (or append(line)).
2. ProgressSession has pending buffer + textStorage flush (not immediate string +=).

## Steps

1. ClientLeaf=enqueue-only.

## Context

REQUIREMENT #8; RED while append immediately mutates textView.string on every line.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "enqueue-only"
	return nil
}
```
