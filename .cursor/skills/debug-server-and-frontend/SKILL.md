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

**Note:** Quick-test mode automatically manages backend and frontend lifecycle, including killing any existing processes on the port.

## Scripts

### debug-server-and-frontend
Starts a quick-test server (port 3580) and opens a browser debugger.

Options:
- `--port PORT` - Port for quick-test server (default: 3580)
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

- Quick-test server runs on port 3580
- Quick-test mode handles process lifecycle (build, start, kill existing)
- Port 3580 is publicly accessible at `https://port-3580-ae2842d.xhd2015.xyz` (equivalent to `localhost:3580`)
- Server exits after **10 minutes of inactivity**
- Server runs from home directory

## Related Code

- Backend: `server/`
- Frontend: `ai-critic-react/`
- Scripts: `script/debug-server-and-frontend/`
- Quick-test lib: `script/lib/quicktest.go`
