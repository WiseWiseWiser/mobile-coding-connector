# Scenario

**Feature**: Collect reads PIDs from opencode-serve-children.json

```
write fixture registry -> CollectOpencodeServePIDs -> returns fixture pid
```

## Preconditions

- Fixture registry written with known PID (current process PID).

## Steps

1. `WriteFixture = true`, `FixturePID` from `os.Getpid()`.

## Context

Unit-level registry read before production launch wiring.

```go
import (
	"os"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.WriteFixture = true
	req.FixturePID = os.Getpid()
	req.FixturePort = 50776
	return nil
}
```
