# Scenario

**Feature**: LaunchCustomAgent writes custom-agent registry entry

```
TestExported_LaunchCustomAgent -> registry kind=custom-agent with pid/port
```

## Preconditions

- Fixture agent `doctest-cleanup-agent` written under temp HOME.

## Steps

1. `Op = OpCustomRegistry`, `CustomAgentID = doctest-cleanup-agent`.

## Context

Validates Path B registration distinct from headless-agent kind.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpCustomRegistry
	req.CustomAgentID = "doctest-cleanup-agent"
	req.UseFakeOpenCode = true
	return nil
}
```
