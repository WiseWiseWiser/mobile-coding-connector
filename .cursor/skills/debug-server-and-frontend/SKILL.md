---
name: debug-server-and-frontend
description: Debug server/frontend integration with quick-test and Playwright
---

# Debug Server and Frontend Skill

## Overview

This skill provides automated debugging for the server and frontend integration using Playwright. It starts a quick-test server and opens a browser debugger.

## Quick Start

```bash
go run ./script/debug-server-and-frontend "await navigate('/project/lifelog-private/home/opencode-web', { waitUntil: 'domcontentloaded' }); console.log('title:', await page.title());"
```

**Note:** Quick-test mode automatically manages backend and frontend lifecycle, including killing any existing processes on the port.

## Scripts

### debug-server-and-frontend
Starts a quick-test server (port 3580) and opens a browser debugger.

Options:
- `--port PORT` - Port for quick-test server (default: 3580)

## Script Variables

| Variable | Description |
|----------|-------------|
| page | Playwright Page object |
| browser | Playwright Browser object |
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

## Quick-Test Endpoints

When running in quick-test mode, the following debug endpoints are available:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/quick-test/health` | GET | Health check with mode info |
| `/api/quick-test/status` | GET | Quick-test mode status |
| `/api/quick-test/config` | GET | Quick-test configuration |
| `/api/quick-test/env` | GET | Environment and system info |
| `/api/quick-test/auth-proxy/status` | GET | Auth proxy status (placeholder) |
| `/api/quick-test/webserver/status` | GET | Webserver status (placeholder) |
| `/api/quick-test/webserver/autostart` | POST | Trigger webserver autostart |

### Example: Testing Quick-Test Endpoints

```bash
# Test health endpoint
curl http://localhost:3580/api/quick-test/health

# Test status endpoint
curl http://localhost:3580/api/quick-test/status

# Trigger webserver autostart (useful for debugging auth proxy issues)
curl -X POST http://localhost:3580/api/quick-test/webserver/autostart
```

### Using Playwright to Test Endpoints

```javascript
// In the debug script, you can use fetch:
const health = await fetch('/api/quick-test/health').then(r => r.json());
console.log('Health:', health);

const status = await fetch('/api/quick-test/status').then(r => r.json());
console.log('Status:', status);

// Trigger autostart and check logs
await fetch('/api/quick-test/webserver/autostart', { method: 'POST' });
```

## Autostart Debugging

If you need to debug why the auth proxy isn't starting:

1. **Trigger autostart manually:**
   ```bash
   curl -X POST http://localhost:3580/api/quick-test/webserver/autostart
   ```

2. **Check server logs** for messages starting with:
   - `[opencode] AutoStartWebServer: BEGIN`
   - `[opencode] AutoStartWebServer: loaded settings`
   - `[opencode] AutoStartWebServer: StartWebServer result`

3. **Common issues:**
   - `no default domain configured, skipping` - Settings missing
   - `failed to ensure extension tunnel configured` - Cloudflare issue

## Related Code

- Backend: `server/`
- Frontend: `ai-critic-react/`
- Scripts: `script/debug-server-and-frontend/`
- Quick-test lib: `script/lib/quicktest.go`
- Quick-test handler: `server/quicktest/handler.go`
