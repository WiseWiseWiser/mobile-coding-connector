# Scenario

**Feature**: Status menu children are only Enable and Disable

```
BackupStatusMenuChildren() -> ["Enable", "Disable"]
```

## Preconditions

Order sealed as Enable then Disable.

## Steps

1. Set `Op=menu_children`.

## Context

REQUIREMENT #21.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "menu_children"
	return nil
}
```
