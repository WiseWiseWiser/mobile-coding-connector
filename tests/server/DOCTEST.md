# Server Doctests

Tests that verify server-side behaviour by starting the actual `ai-critic-server`
binary with a test-controlled configuration directory.

## Decision Tree

```
[ai-critic-server startup]
 |
 +-- Auto-start via opencode settings
      |
      +-- WebServer.Enabled=true at boot
      |    |
      |    +-- AuthProxyEnabled=false
      |    |    |
      |    |    +-- [decision/auto-start-via-settings]  (LEAF)
      |    |         - Writes opencode.json with WebServer.Enabled=true, DefaultDomain set
      |    |         - Starts server in normal mode
      |    |         - Asserts: AutoStartWebServer log messages appear
      |    |         - Asserts: opencode web server port becomes accessible (if binary available)
      |    |
      |    +-- AuthProxyEnabled=true
      |         |
      |         +-- [decision/with-auth-proxy]  (LEAF)
      |              - Writes opencode.json with WebServer.Enabled=true, AuthProxyEnabled=true
      |              - Starts server in normal mode
      |              - Asserts: AutoStartWebServer log messages appear with AuthProxyEnabled=true
      |              - Asserts: [basic_auth_proxy] Proxy started on port log appears
      |              - Asserts: proxy port (14100) is reachable via ProxyRunning
      |              - Asserts: backend port discovered from basic-auth-proxy.json and reachable
      |
      +-- WebServer.Enabled=false at boot
           |
           +-- [decision/disabled-no-autostart]  (LEAF)
           |    - Writes opencode.json with WebServer.Enabled=false, DefaultDomain set
           |    - Starts server in normal mode
           |    - Asserts: NO AutoStartWebServer log messages
           |    - Asserts: web server port is NOT accessible
           |
           +-- [decision/enable-via-api-triggers-start]  (LEAF)
                - Writes opencode.json with WebServer.Enabled=false, DefaultDomain set
                - Starts server in normal mode (no auto-start initially)
                - Makes POST to /api/agents/opencode/settings enabling web server
                - Asserts: initial logs have NO auto-start
                - Asserts: post-API-call logs HAVE auto-start messages
                - Asserts: web server port becomes accessible (if binary available)
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `decision/auto-start-via-settings` | Verifies `AutoStartWebServer()` triggers when settings have `WebServer.Enabled=true` and a valid `DefaultDomain` |
| 2 | `decision/disabled-no-autostart` | Verifies `AutoStartWebServer()` is NOT triggered when `WebServer.Enabled=false` at boot |
| 3 | `decision/enable-via-api-triggers-start` | Verifies that enabling the web server via settings API triggers `AutoStartWebServer()` (bug fix: `handleOpencodeSettings` re-evaluates auto-start after save) |
| 4 | `decision/with-auth-proxy` | Verifies `AutoStartWebServer()` triggers with auth proxy enabled, proxy starts on configured port, backend port is discovered from `basic-auth-proxy.json` |

## Parameter Coverage

| Leaf | Enabled | Domain | Port | AuthProxy | PostStart |
|------|---------|--------|------|-----------|-----------|
| auto-start-via-settings | true | test-auto-start.example.com | 14096 | false | — |
| disabled-no-autostart | false | test-disabled.example.com | 14096 | false | — |
| enable-via-api-triggers-start | false→true | test-enable-via-api.example.com | 14096 | false | POST /settings |
| with-auth-proxy | true | test-auth-proxy.example.com | 14100 | true | — |

## Edge Cases Covered

- Non-localhost domain triggers auto-start (as opposed to localhost which skips)
- Custom port (14096, 14100) instead of default (4096)
- Custom config home directory (via `AI_CRITIC_HOME`)
- Missing opencode binary (handled gracefully)
- Disabled web server at boot (no auto-start)
- API-mediated enable triggers auto-start
- Pre/post API call log segmentation
- Basic auth proxy enabled: proxy wraps web server, backend port discovered from `basic-auth-proxy.json`
- Auth proxy binary built from source during test, prepended to PATH

## How to Run

```sh
doctest test ./tests/server/...
```

Or for a single leaf:

```sh
doctest test ./tests/server/decision/auto-start-via-settings
doctest test ./tests/server/decision/disabled-no-autostart
doctest test ./tests/server/decision/enable-via-api-triggers-start
doctest test ./tests/server/decision/with-auth-proxy
```

Validate tree structure:

```sh
doctest vet ./tests/server
```
