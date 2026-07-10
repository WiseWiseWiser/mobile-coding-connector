# Scenario

**Feature**: remote Cron shows Not configured when endpoint missing

```
remote Menu("Cron") + !configured -> Text("Not configured")
```

## Preconditions

Swift sources for local and/or remote menu-bar apps are present.

## Steps

1. Set `ClientLeaf=remote-not-configured`.

## Context

REQUIREMENT leaf: `client/remote-not-configured`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "remote-not-configured"
	return nil
}
```
