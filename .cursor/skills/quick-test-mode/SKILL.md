---
name: quick-test-mode
description: Run the server in quick-test mode for backend API testing only. Use when the user wants to test backend APIs without starting frontend, health checks, or external tunnel operations.
---

# Quick-Test Mode

## What it does

Quick-test mode runs the server with:
- **NO** auto mapping of domains/exposed URLs
- **NO** health checks for opencode web server or exposed URLs
- **NO** external webserver auto-start
- **NO** Cloudflare tunnel auto-configuration

This is ideal for testing backend API endpoints without side effects.

## Quick Start

```bash
# Run quick-test mode (will start on port 37651 by default)
go run ./script/run quick-test

# Or run directly with the flag
go run . --quick-test
```

The server will:
1. Automatically kill any previous quick-test server on the same port
2. Start on port 37651
3. Skip all auto-start operations
4. **Shut down after 1 minute of inactivity** (or Ctrl+C to stop manually)

## Important Notes

- **Auto-kill previous server**: The script automatically kills any existing quick-test server on the same port before starting a new one.
- **1-minute idle timeout**: The server will automatically exit after 60 seconds without any requests. Keep making requests to keep it alive.
- Use `--port` flag to run on a different port: `go run . --quick-test --port=37652`

## Testing Backend APIs

Once quick-test server is running, test APIs using:

```bash
# Test Cloudflare status
curl http://localhost:37651/api/cloudflare/status

# Test domains
curl http://localhost:37651/api/domains

# Test exposed URLs
curl http://localhost:37651/api/exposed-urls
```

Or use the request script:
```bash
go run ./script/request /api/cloudflare/status
```

## Key Differences from Normal Mode

| Feature | Normal Mode | Quick-Test Mode |
|---------|-------------|-----------------|
| Server port | 23712 | 37651 |
| Auto-start domains | Yes | No |
| Health checks | Yes | No |
| Tunnel config | Auto | No |
| Web server | Auto-start | No |
| Auto-shutdown | No | After 1 min idle |

## When to Use

- Testing Cloudflare authentication status
- Debugging API endpoints
- Testing config changes without affecting running tunnels
- Quick API prototyping
- **Tip**: Keep making requests periodically to avoid the 1-minute idle timeout

## Notes

- The quick-test server shares the same config files (`.ai-critic/`)
- Changes made in quick-test mode will persist (e.g., if you add a domain, it will be saved)
- Use `--port` flag to run on a different port: `go run . --quick-test --port=37652`
