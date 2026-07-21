# Scenario

**Feature**: unknown config flag is rejected

```
# unknown flag -> non-zero error (point at help)
remote-agent config --not-a-real-flag -> error
```

## Preconditions

None.

## Steps

1. Args = `config --not-a-real-flag`.

## Context

T8.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"config", "--not-a-real-flag"}
	return nil
}
```
