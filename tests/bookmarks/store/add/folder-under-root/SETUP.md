# Scenario

**Feature**: add folder under root with empty children

```
Add type=folder name=Dev -> folder child, children empty
```

## Preconditions

1. NodeType folder; no URL required.

## Steps

1. Add folder Dev.
2. Assert type folder and empty children.

## Context

Nested structure container.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.NodeType = "folder"
	req.Name = "Dev"
	return nil
}
```
