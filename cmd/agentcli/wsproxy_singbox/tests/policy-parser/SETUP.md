# Scenario

**Feature**: `ParseDomainPolicy` CLI policy resolution

```
# raw PolicyInput -> validated DomainPolicy or error
ParseDomainPolicy(PolicyInput)
```

## Steps

1. Set `Request.Op = OpParsePolicy`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpParsePolicy
	return nil
}
```