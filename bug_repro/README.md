# Git Fetch Blocking Bug Reproduction

This directory contains a reproduction of the HTTP server blocking issue when running `git fetch` without an SSH key.

## Overview

This reproduction uses the project's existing packages:
- `server/gitrunner` - For git command execution with proper environment setup
- `server/sse` - For Server-Sent Events streaming
- `server/cloudflare` - For Cloudflare tunnel management (like the main server)

## Files

- `main.go` - HTTP server that replicates the main server's behavior including:
  - Git operations with SSE streaming
  - Auto-starting Cloudflare tunnel on server startup
- `bug_server` - Compiled binary

## Running

```bash
cd /root/mobile-coding-connector
./bug_repro/bug_server
```

Or build from source:
```bash
go build -o bug_repro/bug_server ./bug_repro/main.go
./bug_repro/bug_server
```

## Cloudflare Tunnel

The server automatically starts a Cloudflare tunnel on startup (like the main server):
- **Domain**: `test-bug-hang.xhd2015.xyz`
- **Port**: 8080
- **Behavior**: Runs in a goroutine, same as `domains.AutoStartTunnels()` in the main server

The tunnel is started using `cloudflareSettings.StartDomainTunnel()` which:
1. Finds or creates a named tunnel
2. Creates a DNS route for the domain
3. Starts the `cloudflared` process
4. Routes traffic from the domain to `http://localhost:8080`

## Test Endpoints

### Health Check
```bash
curl http://localhost:8080/
```
Returns: `OK #N - <timestamp>`

### Git Fetch with SSE Streaming (like real server)
```bash
curl -X POST http://localhost:8080/api/git/fetch
```
Uses the same SSE streaming as the main server (`server/github/gitops.go`).

### Direct Git Fetch
```bash
curl http://localhost:8080/fetch-direct
```
Uses `gitrunner.Fetch().Dir(repo).Run()` directly.

### Raw Git Fetch (without gitrunner)
```bash
curl http://localhost:8080/fetch-raw
```
Uses raw `exec.Command` without gitrunner's environment setup.

### Multiple Concurrent Fetches
```bash
curl http://localhost:8080/fetch-many
```
Launches 5 concurrent git fetches in the background.

## Observed Behavior

In our testing environment:
1. Git fetch fails immediately with "Host key verification failed" 
2. The error is properly returned via both SSE and direct responses
3. The HTTP server remains responsive to concurrent requests
4. No blocking is observed

## Expected vs Actual

**Expected (bug report):** When git fetch runs without SSH key, the HTTP server should block and all subsequent requests should hang.

**Actual (observed):** Git fetch fails immediately with SSH error, server remains responsive.

## Possible Explanations

1. **Environment differences** - The bug may only occur in specific SSH configurations
2. **SSH client behavior** - Different SSH versions may handle missing keys differently
3. **Git configuration** - Certain git configs might cause different behavior
4. **Go version** - Go's exec behavior might differ between versions

## Debug Information

To debug further:

1. Check SSH configuration:
```bash
ssh -V
cat ~/.ssh/config
```

2. Check git configuration:
```bash
git config --list
```

3. Test SSH behavior:
```bash
GIT_SSH_COMMAND="ssh -v" git -C /root/lifelog-private fetch origin 2>&1
```

4. Check if SSH agent is running:
```bash
ssh-add -l
echo $SSH_AUTH_SOCK
```

## Related Code

The reproduction uses the same patterns as the main server:

- `server/github/gitops.go:53-144` - `runGitOp` function that handles git fetch/pull/push with SSE
- `server/sse/sse.go:60-104` - `StreamCmd` function for streaming command output
- `server/gitrunner/gitrunner.go:106-116` - `Run` and `Output` methods
- `server/cloudflare/domain_tunnel.go:93-152` - `StartDomainTunnel` function for starting Cloudflare tunnels
- `server/domains/domains.go:111-145` - `AutoStartTunnels` function that starts tunnels on server startup
