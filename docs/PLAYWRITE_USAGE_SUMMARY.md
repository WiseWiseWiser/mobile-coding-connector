# Playwriter Usage Summary

Playwriter is a Chrome extension + CLI that lets AI agents use your real browser via Chrome DevTools Protocol (CDP). No Chrome restart or special flags needed.

## Setup

### 1. Install the Chrome extension

Install from the Chrome Web Store: [Playwriter MCP](https://chromewebstore.google.com/detail/playwriter-mcp/jfeammnjpkecdekppnclgkkfahahfhe)

### 2. Start the relay server

For local use (no token required):

```bash
npx playwriter@latest serve --host localhost
```

For remote/LAN access (requires token):

```bash
npx playwriter@latest serve --host 0.0.0.0 --token <your-token>
```

> **Note:** When using `--host 0.0.0.0` with a token, the CLI's WebSocket connection may fail with 401 (bug in v0.0.80). Use `--host localhost` for local testing.

### 3. Activate on a tab

Click the Playwriter extension icon on the tab you want to control. The icon turns green when connected.

## Usage

### Create a session

```bash
npx playwriter@latest session new
```

Output:

```
Session 1 created. Use with: playwriter -s 1 -e "..."
```

### Execute commands

Get page title:

```bash
npx playwriter@latest -s 1 -e "return await page.title()"
# [return value] ai-critic
```

Get current URL:

```bash
npx playwriter@latest -s 1 -e "return await page.url()"
# [return value] http://localhost:3580/home/acp/cursor
```

Navigate to a page:

```bash
npx playwriter@latest -s 1 -e "await page.goto('http://localhost:3580/home/tools')"
```

Get page content:

```bash
npx playwriter@latest -s 1 -e "return await page.content()"
```

Take a snapshot:

```bash
npx playwriter@latest -s 1 -e "return await snapshot({ page })"
```

Click an element:

```bash
npx playwriter@latest -s 1 -e "await page.locator('text=Install').first().click()"
```

### Session management

```bash
npx playwriter@latest session list              # list active sessions
npx playwriter@latest session delete <id>       # delete a session
npx playwriter@latest session reset <id>        # reset browser connection
```

## Troubleshooting

### Relay restarts between CLI invocations

If the relay restarts and loses sessions, chain commands:

```bash
npx playwriter@latest session new && npx playwriter@latest -s 1 -e "return await page.title()"
```

### Token auth issues with --host 0.0.0.0

The CLI (v0.0.80) may fail to pass the token via WebSocket. Workaround: use `--host localhost` for local testing.

### HTTP API (curl)

When using `--host 0.0.0.0 --token <token>`, the REST API accepts the token as a query parameter:

```bash
# List sessions
curl -s "http://localhost:19988/cli/sessions?token=<token>"

# Create session
curl -s -X POST -H "Content-Type: application/json" -d '{}' \
  "http://localhost:19988/cli/session/new?token=<token>"
```

## Architecture

- Chrome extension uses `chrome.debugger` API to attach to tabs
- Extension connects to local relay server via WebSocket (`ws://localhost:19988/extension`)
- CLI/agents connect via WebSocket (`ws://localhost:19988/cdp/<session-id>`)
- CDP commands flow: CLI -> relay -> extension -> Chrome -> extension -> relay -> CLI
- Everything runs locally; nothing leaves your machine unless explicitly sent

## Log files

```bash
npx playwriter@latest logfile
# relay: ~/.playwriter/relay-server.log
# cdp: ~/.playwriter/cdp.jsonl
```
