# Scenario

**Feature**: per-service submenu title strings

```
name + status + enabled -> FormatServiceTitle -> title line
```

## Preconditions

`Op=title` dispatches to `menubar.FormatServiceTitle`.

## Steps

1. Leaf supplies `Name`, `Status`, and `Enabled`.

## Context

REQUIREMENT section A — service menu title formatters.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "title"
	return nil
}
```