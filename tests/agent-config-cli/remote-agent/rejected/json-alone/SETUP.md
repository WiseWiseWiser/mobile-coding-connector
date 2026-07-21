# Scenario

**Feature**: --json without --show is an error

```
# --json alone requires --show
remote-agent config --json -> non-zero, message mentions --show
```

## Preconditions

None.

## Steps

1. Args = `config --json`.

## Context

T6.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"config", "--json"}
	return nil
}
```
