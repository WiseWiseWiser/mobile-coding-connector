# Scenario

**Feature**: permanent 502 exhausts three attempts

```
GET x3 -> error; file incomplete or absent
```

## Preconditions

Inherited from `retry-exhaustion/SETUP.md`.

## Steps

No additional setup.

## Context

Leaf asserts attempt cap and failure.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if !req.AlwaysFail || req.MaxDownloadAttempts != 3 {
		t.Fatalf("parent setup: AlwaysFail=%v MaxDownloadAttempts=%d, want true/3", req.AlwaysFail, req.MaxDownloadAttempts)
	}
	return nil
}
```