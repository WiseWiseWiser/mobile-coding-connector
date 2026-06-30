# Scenario

**Feature**: invalid project directory rejected at launch

```
launch(grok, file-not-dir) -> invalid project directory error
```

## Preconditions

- `InvalidProject = true` uses a temp file path.

## Steps

1. `InvalidProject = true`, `UseFakeOpenCode = true` (binary present; dir validation fails first).

## Context

Directory stat must fail before or independent of binary.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpLaunchGrok
	req.InvalidProject = true
	req.UseFakeOpenCode = true
	return nil
}
```