# Scenario

**Feature**: credentials file with only blank lines → none

```
server-credentials = "\n  \n\t\n" (config missing)
  -> token="", source=none
```

## Preconditions

Credentials file exists but every line is empty after trim; config missing.

## Steps

1. Omit config.
2. Write credentials consisting only of blank/whitespace lines.

## Context

REQUIREMENT edge of scenario 6/7: blank credentials do not invent a token.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigPresent = false
	req.CredentialsPresent = true
	req.CredentialsText = "\n  \n\t\n \n"
	return nil
}
```
