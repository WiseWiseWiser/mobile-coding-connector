# Scenario

**Feature**: residual update banner alone must not block idle when model is not loading

```
04-idle-prompt.snapshot.txt with model:loading stripped -> IsBlocking=false, writable idle
```

## Preconditions

Base fixture `04-idle-prompt.snapshot.txt` (banner + main `›` prompt). Harness sets
`StripModelLoading=true` so the only residual update signal is the banner.

## Steps

1. `FixtureFile=04-idle-prompt.snapshot.txt`.
2. `StripModelLoading=true`.

## Context

Documents PROTOCOL note: banner is non-blocking; still honor real `model:loading`
(covered by menu-dismissed leaf without strip).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "04-idle-prompt.snapshot.txt"
	req.StripModelLoading = true
	return nil
}
```
