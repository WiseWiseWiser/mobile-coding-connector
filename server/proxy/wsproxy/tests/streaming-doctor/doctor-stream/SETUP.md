# Scenario

**Feature**: Manager.DoctorStream emits incremental server checks

```
# doctor runs serverDoctorChecks with per-check emit callback
Manager.DoctorStream -> progress events (layer=server) -> done
```

## Preconditions

`Request.Target` is `doctor-stream`. Fake xray and stubbed network checks keep
the stream fast and deterministic.

## Steps

1. Set `req.Target = TargetDoctorStream`.
2. Enable `SimulateXray` and `StubNetworkChecks` for all doctor-stream leaves.

## Context

Exercises the first consumer of `progress.Writer` — ws-proxy doctor streaming.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Target = TargetDoctorStream
	req.SimulateXray = true
	req.StubNetworkChecks = true
	return nil
}
```
