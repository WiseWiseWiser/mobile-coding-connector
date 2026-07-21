# Scenario

**Feature**: --show with no config file

```
# missing remote-agent-config.json -> empty-ish pretty JSON, exit 0
remote-agent config --show -> {"domains": ... empty ...}
```

## Preconditions

Do not seed config; file must not exist.

## Steps

1. Args = `config --show`.
2. SeedConfig = nil.

## Context

T3; spirit of GET /api/config when nil.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"config", "--show"}
	req.SeedConfig = nil
	return nil
}
```
