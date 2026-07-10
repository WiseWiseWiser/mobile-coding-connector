# Scenario

**Feature**: config unusable → fall through to server-credentials

```
# config missing | empty token | invalid JSON
local-agent-config.json (fail) -> server-credentials first non-empty line
  -> token from credentials, source=credentials
```

## Preconditions

Config path fails or yields no non-empty trimmed token; credentials file has at
least one non-empty line (except when a leaf is testing blank-line selection with
a real token after blanks).

## Steps

1. Leaf configures config failure mode and seeds credentials.
2. Expect `source=credentials`.

## Context

REQUIREMENT group: `resolve/fallthrough-credentials/` (scenarios 2–4, 7).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CredentialsPresent = true
	return nil
}
```
