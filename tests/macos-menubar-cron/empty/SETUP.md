# Scenario

**Feature**: Cron menu empty-list and not-configured placeholders

```
empty task list -> FormatCronTasksEmptyLabel
remote missing endpoint -> FormatCronNotConfiguredLabel
```

## Preconditions

`Op=empty` dispatches to empty / not-configured label helpers.

## Steps

1. Leaf sets `NotConfigured` when testing remote missing endpoint.

## Context

REQUIREMENT empty: `No cron tasks configured`; remote: `Not configured`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "empty"
	return nil
}
```
