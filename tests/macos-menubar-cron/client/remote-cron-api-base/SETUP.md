# Scenario

**Feature**: remote cron APIs use configured base URL + auth, not keep-alive

```
ServiceClient/baseURL + Bearer -> /api/cron-tasks*  (not :23312 keep-alive)
```

## Preconditions

Swift sources for local and/or remote menu-bar apps are present.

## Steps

1. Set `ClientLeaf=remote-cron-api-base`.

## Context

REQUIREMENT leaf: `client/remote-cron-api-base`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "remote-cron-api-base"
	return nil
}
```
