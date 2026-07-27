# Scenario

**Feature**: ai-critic-server auto-start and settings-driven web server behaviour

```
opencode.json settings -> start ai-critic-server -> AutoStartWebServer logs / ports
```

## Preconditions

A temporary directory is used as the server config home. The server reads all
configuration from this directory via the `AI_CRITIC_HOME` environment variable,
instead of the default `~/.ai-critic` directory. Test credentials use username
`test` (convention only) and password/token `testpassword` written to
`server-credentials`.

## Steps

1. Create a temporary config home directory
2. Store path on `Request.ConfigHome` (child server gets `AI_CRITIC_HOME` only via `cmd.Env`)
3. Write test-specific config files (`opencode.json`, etc.) into the config home
4. Build the server binary (`go build -o <tmp>/ai-critic-server .`) and the basic-auth-proxy binary (`go build -o <tmp>/basic-auth-proxy ./cmd/basic-auth-proxy`)
5. Start the server on a test-controlled port in normal (non-quick-test) mode
6. Wait for the server to become ready (`GET /ping` responds)
7. Capture server stdout/stderr for log verification
8. If `AuthProxyEnabled`: read `basic-auth-proxy.json` to discover the backend port, then check proxy and backend port reachability
9. After the test completes, stop the server process and clean up the temp directory

## Context

The root `Run` function is the test entry point. It accepts a `Request` describing
the desired test scenario and returns a `Response` with collected data for
assertion. The `Request.OpenCodeSettings` controls what is written to
`opencode.json` before the server starts.

This directory tests the server's behaviour when started with various opencode
settings configurations. The key behaviours under test are:

- Whether `AutoStartWebServer()` is triggered (log messages appear)
- Whether the opencode web server starts on the configured port
- Whether the basic-auth-proxy starts and proxies correctly
- How the server handles missing `opencode` binary

### Parameters (ranked by significance)

| # | Parameter | Type | Values | Description |
|---|-----------|------|--------|-------------|
| 1 | `WebServer.Enabled` | bool | true, false | Whether auto-start should trigger |
| 2 | `DefaultDomain` | string | valid domain, "", localhost | Domain for tunnel mapping; localhost skips |
| 3 | `WebServer.Port` | int | 1-65535, 0 (default=4096) | Port for opencode web server |
| 4 | `AuthProxyEnabled` | bool | true, false | Whether basic-auth-proxy wraps the web server |
| 5 | `opencode` binary | availability | present, absent | Whether the opencode CLI is in PATH |
| 6 | `basic-auth-proxy` binary | availability | present, absent | Whether the proxy binary is in PATH (built by test) |
| 7 | `AI_CRITIC_HOME` | env var | path | Custom config home directory |
