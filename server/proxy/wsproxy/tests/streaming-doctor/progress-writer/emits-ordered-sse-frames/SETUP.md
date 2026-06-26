# Scenario

**Feature**: progress.Writer emits frames in invocation order

```
# emit 2 progress, 1 section, 1 done
EmitProgress -> EmitProgress -> EmitSection -> EmitDone
```

## Preconditions

Inherited `TargetProgressWriter` from ancestor.

## Steps

No additional setup — root `Run` emits the canonical four-frame sequence.

## Context

Maps to requirement scenario `progress-writer-framework`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Target = TargetProgressWriter
	return nil
}
```
