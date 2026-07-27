# Scenario

**Feature**: --show --json is a no-op relative to --show

```
# --show --json succeeds with same config dump as --show
remote-agent config --show --json -> pretty JSON (seeded)
```

## Preconditions

Same seed as with-domains.

## Steps

1. Seed sampleRemoteConfig().
2. Args = `config --show --json`.

## Context

T5; --json alone is rejected elsewhere.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedConfig = sampleRemoteConfig()
	req.Args = []string{"config", "--show", "--json"}
	return nil
}
```
