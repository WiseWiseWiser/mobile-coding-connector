# Debug Server and Frontend

A debugging skill using Puppeteer to debug and verify server and frontend integration.

## Purpose

This skill provides browser automation for debugging the quick-test server and frontend pages.

## Usage

### Quick Start

```bash
# Start quick-test server and debug interactively
go run ./script/debug-server-and-frontend

# Debug with visible browser
go run ./script/debug-server-and-frontend --no-headless
```

**Note:** This script always kills any existing server on the port and builds/deploys the latest code, ensuring you're testing the most recent changes.

### Script Examples

```bash
# Get page title
printf "console.log('Title:', await page.title())" | go run ./script/debug-server-and-frontend

# Navigate to a path
printf "await navigate('/mockups/path-input'); console.log(await page.title())" | go run ./script/debug-server-and-frontend

# Take screenshot
printf "await navigate('/'); const buf = await page.screenshot(); fs.writeFileSync('screenshot.png', buf)" | go run ./script/debug-server-and-frontend
```

## Server Behavior

- **Always fresh**: Script kills any existing server and builds/deploys the latest code
- Quick-test server runs on port 3580
- Port 3580 is publicly accessible at `https://port-3580-ae2842d.xhd2015.xyz` (equivalent to `localhost:3580`)
- Server exits after **10 minutes of inactivity**
- Server runs from home directory

## Script Variables

| Variable | Description |
|----------|-------------|
| page | Puppeteer Page object |
| browser | Puppeteer Browser object |
| console | Node console |
| fs | Node fs module |
| BASE_URL | Base URL (http://localhost:{port}) |
| navigate(url) | Navigate helper (auto-prepends BASE_URL) |

## Options

### debug-server-and-frontend
- `--port PORT` - Port for server (default: 3580)
- `--no-headless` - Run with visible browser

## Related Files

- `script/debug-server-and-frontend/main.go` - Server + debug runner
- `script/debug-port/debug.js` - Puppeteer automation
