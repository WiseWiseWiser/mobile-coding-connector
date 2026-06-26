# Scenario

**Feature**: terminal done frame carries aggregate doctor summary

```
# after all progress events, done includes healthy + counts
serverDoctorChecks -> done { healthy, try_url, checks_total, checks_failed }
```

## Preconditions

Standard doctor-stream fixture (fake xray, stubbed network).

## Steps

Set explicit `TryURL` for assertion.

## Context

Extended coverage for the `done` envelope fields defined in the requirement.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.TryURL = "https://example.com/doctor-test"
	return nil
}
```
