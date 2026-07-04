# Scenario

**Feature**: FormatGrokLabel by usage status

```
FormatGrokLabel(status, weeklyLimit, errorMsg) -> menu bar label
```

## Preconditions

Truncation budget comes from `menubar.TestExported_MaxLabelLen()` in `Run`.

## Steps

1. Leaf setup supplies status-specific inputs.

## Context

All label variants under one grouping factor: `status`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Leaves supply status-specific inputs; reset shared label fields first.
	req.Status = ""
	req.WeeklyLimit = ""
	req.ErrorMsg = ""
	return nil
}
```