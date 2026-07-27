# Scenario

**Feature**: --show with domains and default

```
# seeded remote-agent-config.json -> pretty JSON matches content (tokens full)
remote-agent config --show -> stdout JSON == seed
```

## Preconditions

Seed multi-domain config with full tokens.

## Steps

1. Seed sampleRemoteConfig().
2. Args = `config --show`.

## Context

T4.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedConfig = sampleRemoteConfig()
	req.Args = []string{"config", "--show"}
	return nil
}
```
