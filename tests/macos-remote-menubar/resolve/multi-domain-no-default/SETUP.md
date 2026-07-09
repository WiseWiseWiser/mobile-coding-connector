# Scenario

**Feature**: multiple domains with empty default → no_default

```
default="" + two domains -> state=no_default, no endpoint
```

## Preconditions

Spec: `no_default` when >1 domain and no usable default match (not first-domain).

## Steps

1. Set two domains and empty default.

## Context

REQUIREMENT leaf: `resolve/multi-domain-no-default`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "",
  "domains": [
    {"server": "https://a.example", "token": "a"},
    {"server": "https://b.example", "token": "b"}
  ]
}`
	return nil
}
```
