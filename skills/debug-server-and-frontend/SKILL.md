# Debug Server and Frontend Skill

## Overview

This skill provides automated debugging for the server and frontend integration using Puppeteer. It starts a quick-test server and opens a browser debugger.

## Quick Start

```bash
# Start server and debug interactively
go run ./script/debug-server-and-frontend

# Run with visible browser
go run ./script/debug-server-and-frontend --no-headless
```

**Note:** This script always kills any existing server on the port and deploys the latest code, ensuring you're testing the most recent changes.

## Scripts

### debug-server-and-frontend
Starts a quick-test server (port 37651) and opens a browser debugger. Kills any existing server on the port first to ensure an up-to-date server.

Options:
- `--port PORT` - Port for quick-test server (default: 37651)
- `--no-headless` - Run browser with visible window

## Script Variables

| Variable | Description |
|----------|-------------|
| page | Puppeteer Page object |
| browser | Puppeteer Browser object |
| console | Node console |
| fs | Node fs module |
| BASE_URL | Base URL string (http://localhost:{port}) |
| VIEWPORT_WIDTH | Viewport width (default: 375) |
| VIEWPORT_HEIGHT | Viewport height (default: 800) |
| navigate(url, opts) | Navigate helper (auto-prepends BASE_URL) |

## Server Behavior

- **Always fresh**: Script kills any existing server and builds/deploys the latest code
- Quick-test server runs on port 37651
- Port 37651 is publicly accessible at `https://port-37651-ae2842d.xhd2015.xyz` (equivalent to `localhost:37651`)
- Server exits after **10 minutes of inactivity**
- Server runs from home directory

## Related Code

- Backend: `server/`
- Frontend: `ai-critic-react/`
- Scripts: `script/debug-server-and-frontend/`
