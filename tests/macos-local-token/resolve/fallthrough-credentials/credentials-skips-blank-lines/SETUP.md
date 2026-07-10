# Scenario

**Feature**: credentials uses first non-empty line after blanks

```
server-credentials:
  <blank>
  <spaces>
  first-non-empty-cred
  second-token-ignored
  -> token=first-non-empty-cred, source=credentials
```

## Preconditions

Config missing so credentials path is used; file starts with blank/whitespace lines.

## Steps

1. Omit config.
2. Write credentials with leading blanks, then two tokens.

## Context

REQUIREMENT leaf: scenario 7 (and scenario 2 base path).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigPresent = false
	req.CredentialsText = "\n\n  \n\t\nfirst-non-empty-cred\nsecond-token-ignored\n"
	return nil
}
```
