# Scenario

**Feature**: no_default status guides user to pick default via Configure…

```
FormatStatus(no_default, "") -> multi-server Configure… copy
```

## Preconditions

Multiple domains, no usable default.

## Steps

1. Set `ConnectionState=no_default`.

## Context

REQUIREMENT leaf: `status/no-default`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ConnectionState = "no_default"
	req.StatusServer = ""
	req.Token = "must-not-appear"
	return nil
}
```
