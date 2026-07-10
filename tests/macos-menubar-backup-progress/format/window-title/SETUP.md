# Scenario

**Feature**: progress window title Backup: {server}

```
FormatBackupProgressWindowTitle(name) -> "Backup: {name}" | "Backup: (no server)"
```

## Preconditions

`Op=format_window_title`. Empty name uses sealed placeholder `(no server)`.

## Steps

1. Leaf sets ServerName.

## Context

REQUIREMENT Window section.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_window_title"
	return nil
}
```
