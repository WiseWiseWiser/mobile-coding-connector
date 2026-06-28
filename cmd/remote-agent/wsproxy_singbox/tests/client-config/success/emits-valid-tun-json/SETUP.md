# Scenario

**Feature**: client-config prints valid TUN JSON to stdout

```
# no --output: rendered config goes to stdout
BuildSingBoxTunConfig -> stdout (JSON)
```

## Steps

1. Leave `OutputFile` empty (default stdout).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.OutputFile = ""
	return nil
}
```