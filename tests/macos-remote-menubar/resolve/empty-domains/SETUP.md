# Scenario

**Feature**: empty domains list is not_configured

```
{"default":"","domains":[]} -> Resolve -> not_configured
```

## Preconditions

Config JSON present with empty `domains` array.

## Steps

1. Set `ConfigJSON` with empty domains.

## Context

REQUIREMENT leaf: `resolve/empty-domains`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ConfigJSON = `{"default":"","domains":[]}`
	return nil
}
```
