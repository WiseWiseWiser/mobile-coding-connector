# Scenario

**Feature**: first non-empty local credential line is saved for resolved local server

```
# blank lines before token are ignored
~/.ai-critic/server-credentials -> first-non-empty token

# resolved default localhost server becomes default domain
local-agent auth import-local -> local-agent-config.json default/domains
```

## Preconditions

No server needs to be contacted; reachability is mocked up in case the helper resolves through
the same local profile path as API commands.

## Steps

1. Seed `~/.ai-critic/server-credentials` with blank lines, `local-import-token`, then another token.
2. Seed `local-agent-config.json` with an unrelated domain row and a project binding.
3. Inject default port `24888`, so the resolved local server is `http://localhost:24888`.
4. Run `auth import-local`.

## Context

This leaf asserts token selection, upsert behavior, default selection, preservation of
unrelated config data, and non-disclosure of the raw token.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	up := true
	req.MockReachability = &up
	req.InjectedDefaultPort = 24888
	req.ServerCredentialContent = "\n\n  \nlocal-import-token\nsecond-token-ignored\n"
	req.SeedLocalConfig = &LocalAgentConfigFile{
		Default: "https://old.example.com",
		Domains: []DomainEntry{
			{Server: "https://old.example.com", Token: "old-token"},
		},
		ProjectBindings: []ProjectBinding{
			{Server: "https://old.example.com", RemoteDir: "/remote/project", LocalPath: "/local/project"},
		},
	}
	return nil
}
```
