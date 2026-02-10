# AI Critic

AI Critic is a developer tool that combines AI-powered code review with a mobile-friendly coding workspace. It provides a Go backend server with a React frontend, designed to be accessible remotely via Cloudflare tunnels or other port-forwarding providers.

## What This Project Does

### AI Code Review
- Uses OpenAI (or compatible) APIs to review code changes (git diffs) against configurable review rules
- Supports reviewing uncommitted changes, staged changes, or comparing specific commits
- Review rules are defined in `rules2/REVIEW_RULES.md`

### Mobile Coding Workspace (V2)
- A progressive web app (PWA) optimized for iPhone Safari / mobile use
- Provides a tabbed interface with: Home, Chat, Terminal, Ports, and Files tabs
- Can be added to the home screen for a native app experience

### Port Forwarding
- Expose local development ports to the internet through multiple tunnel providers:
  - **localtunnel**: Free tunneling via `npx localtunnel` (default)
  - **Cloudflare Quick Tunnel**: Free tunneling via `cloudflared` (trycloudflare.com, no account needed)
  - **Cloudflare Named Tunnel**: Custom domain tunneling via a dedicated named Cloudflare tunnel with random subdomain generation
- Frontend UI to add/remove port forwards, view logs, and diagnose issues
- Backend manages tunnel processes and provides real-time status via REST API

### Terminal
- Web-based terminal access to the server's host machine

## Architecture

- **Backend**: Go server (`server/`) with REST API endpoints
  - `server/server.go` - Main HTTP server, route registration
  - `server/api_review.go` - AI code review endpoints
  - `server/portforward/` - Port forwarding manager with provider interface
  - `server/portforward/providers/cloudflare/` - Cloudflare tunnel providers
  - `server/portforward/providers/localtunnel/` - Localtunnel provider
  - `server/terminal/` - Terminal WebSocket handler
  - `server/config/` - Configuration (JSON-based, loaded from `.config.local.json`)
- **Frontend**: React + TypeScript (`ai-critic-react/`)
  - `src/v2/MobileCodingConnector.tsx` - Main V2 mobile workspace component
  - `src/hooks/usePortForwards.ts` - React hook for port forwarding API
  - Uses React Router for navigation, URL search params for tab/view routing
- **Entry point**: `main.go` embeds the built frontend and starts the server
- **Scripts**: `script/` contains build, run, and setup utilities

## Configuration

Configuration is loaded from `.config.local.json` in the project root. Key sections:
- `openai_api_key` / `openai_base_url` - AI provider settings
- `port_forwarding.providers` - Array of tunnel provider configs (type: `localtunnel`, `cloudflare_quick`, `cloudflare_tunnel`)

## Development

```bash
# Run server in development mode
go run ./script/server/run

# Run frontend in development mode
go run ./script/vite/run

# Build for production
go run ./script/build
```

The server listens on port 23712 by default. In dev mode, the frontend is proxied from the Vite dev server (port 5173).

## Coding Conventions

- Go backend follows standard Go project layout
- React components use functional components with hooks
- CSS uses BEM-like naming with `.mcc-` prefix for the V2 mobile workspace
- Port forwarding providers implement the `portforward.Provider` interface
- Configuration is JSON-based with `json` struct tags

## Executing External Commands (tool_exec)

When executing external commands (like `opencode`, `claude`, etc.), **always use the `tool_exec` package** instead of `os/exec` directly. This ensures:

1. **Proper PATH resolution**: Searches extra install directories like `~/.local/bin`, `~/.opencode/bin`, npm global bin, etc.
2. **User-configured paths**: Respects binary paths configured in agent settings
3. **Consistent environment**: All commands run with the same extended PATH

### Usage

```go
import "github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"

// Basic usage - automatic PATH resolution
cmd, err := tool_exec.New("opencode", []string{"web", "start"}, nil)
if err != nil {
    return err
}
output, err := cmd.CombinedOutput()

// With user-configured path (from agent settings)
customPath := agents.GetAgentBinaryPath("opencode")
cmd, err := tool_exec.New("opencode", []string{"web", "start"}, &tool_exec.Options{
    CustomPath: customPath,
})

// With custom environment variables
cmd, err := tool_exec.New("opencode", []string{"web", "start"}, &tool_exec.Options{
    Env: map[string]string{"FOO": "bar"},
})

// With working directory
cmd, err := tool_exec.New("opencode", []string{"web", "start"}, &tool_exec.Options{
    Dir: "/path/to/project",
})
```

### Key Features

- **CustomPath**: If provided, takes priority over PATH resolution. Falls back to PATH search if invalid.
- **Env**: Additional environment variables merged with system environment
- **Dir**: Working directory for the command
- The command inherits the current process environment with extended PATH including:
  - `/usr/local/bin`, `/usr/local/go/bin`
  - `~/.local/bin`, `~/.opencode/bin`, `~/go/bin`
  - npm global bin directory
  - User-configured extra paths from terminal config
