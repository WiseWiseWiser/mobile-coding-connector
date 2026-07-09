# Scenario

**Feature**: selecting domain B updates default and resolve target

```
default=https://a.example + domains[A,B] -> SelectDefaultDomain(https://b.example)
  -> Save -> Load: default=https://b.example, Resolve token=tok-b
```

## Preconditions

Two domains A and B; initial default is A; user selects B’s server URL.

## Steps

1. Seed ConfigJSON with A as default and both domains.
2. Set `SelectServer` to B’s server.

## Context

REQUIREMENT leaf: `domain/select-persists-default` (also covers same-endpoint for services/terminals via shared Resolve).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigJSON = `{
  "default": "https://a.example",
  "domains": [
    {"server": "https://a.example", "token": "tok-a"},
    {"server": "https://b.example", "token": "tok-b"}
  ]
}`
	req.SelectServer = "https://b.example"
	return nil
}
```
