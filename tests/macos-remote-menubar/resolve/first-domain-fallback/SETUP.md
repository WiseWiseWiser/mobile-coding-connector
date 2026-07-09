# Scenario

**Feature**: empty default with exactly one domain uses that domain

```
default="" + one domain -> Resolve that domain, state=ok
```

## Preconditions

Prefer: default match, else if exactly one domain use it.

## Steps

1. Set empty default and a single domain.

## Context

REQUIREMENT leaf: `resolve/first-domain-fallback`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "",
  "domains": [
    {"server": "https://only.example", "token": "only-tok"}
  ]
}`
	return nil
}
```
