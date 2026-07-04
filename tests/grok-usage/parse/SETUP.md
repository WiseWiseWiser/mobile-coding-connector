# Scenario

**Feature**: tty.ParseShowUsageOutput from fixture scrollback

```
fixture scrollback -> tty.ParseShowUsageOutput -> UsageInfo or error
```

## Preconditions

Fixture files under shared `testdata/`. Parser source of truth is `agent/grok/tty`.

## Steps

1. Set `Op=parse` in leaf setup.

## Context

Pure parser tests; no daemon, network, or PTY fetch.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "parse"
	return nil
}
```