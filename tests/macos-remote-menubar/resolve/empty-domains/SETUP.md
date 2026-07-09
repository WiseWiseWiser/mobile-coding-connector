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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{"default":"","domains":[]}`
	return nil
}
```
