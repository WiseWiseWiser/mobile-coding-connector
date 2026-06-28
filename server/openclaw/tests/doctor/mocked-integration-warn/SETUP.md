# Scenario

**Feature**: doctor always warns integration is mocked

```
# mock_mode check is always warn
Manager.Doctor -> check mock_mode (warn)
```

## Steps

1. Default config.
2. Run doctor.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpDoctor
	return nil
}
```