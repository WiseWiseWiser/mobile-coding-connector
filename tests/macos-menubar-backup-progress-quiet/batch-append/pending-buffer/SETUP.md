# Scenario

**Feature**: append enqueues into a pending buffer; flush drains it

```
ProgressSession.append(line) -> pendingLines / buffer
ProgressSession.flush*       -> drain buffer to UI
```

## Preconditions

Symbols such as `pendingLines`, `pending`, `lineBuffer`, `flushPending` /
`flushBuffer` / `func flush` appear; append path references pending/buffer.

## Steps

1. ClientLeaf=pending-buffer.

## Context

REQUIREMENT #5; RED on current direct `appendOnMain` without buffer.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "pending-buffer"
	return nil
}
```
