# Scenario

**Feature**: remote-agent config -h

```
# -h -> same help family
remote-agent config -h -> stdout help, exit 0
```

## Preconditions

None beyond remote profile.

## Steps

1. Args = `config -h`.

## Context

T2 short form.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"config", "-h"}
	return nil
}
```
