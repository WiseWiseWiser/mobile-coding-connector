# Exec-Restart Bug Summary

## Issue Overview
The exec-restart functionality experiences issues during server process replacement, particularly when health checks or port binding conflicts occur during the restart sequence.

## Affected Components

### Primary Files
1. **`/root/mobile-coding-connector/server/server.go`**
   - Main exec-restart endpoint handlers
   - Graceful shutdown coordination
   - Process replacement logic

2. **`/root/mobile-coding-connector/run/daemon/health.go`**
   - Health check monitoring
   - Exec-restart detection and pausing
   - Binary upgrade detection

3. **`/root/mobile-coding-connector/run/daemon/daemon.go`**
   - Keep-alive daemon management
   - Port binding conflict detection
   - Post-restart health check coordination

### Reproduction Scripts
- **`/root/mobile-coding-connector/script/replicate-exec-bug/main.go`**
- **`/root/mobile-coding-connector/script/replicate-exec-bug-fixed/main.go`**
- **`/root/mobile-coding-connector/script/replicate-exec-bug-proper/main.go`**

## Bug Description

### Symptoms
1. **Port Binding Conflicts**: Server fails to restart when port is already in use
2. **Health Check Failures**: Post-restart health checks fail due to timing issues
3. **Process Orphaning**: Old server process not properly terminated before new process starts
4. **Flag Parsing Issues**: Command-line flags not correctly passed during exec restart

### Root Causes

#### 1. Race Condition in Port Binding
```go
// In daemon.go - Race condition between shutdown and bind
if isPortInUse(d.port) {
    // Server may still be shutting down
    // Port check passes but bind fails
}
```

#### 2. Insufficient Health Check Pause
```go
// In health.go - 1 minute pause may be insufficient
HealthCheckPauseDelay  = 1 * time.Minute
// Server may need more time to stabilize
```

#### 3. Improper Flag Passing in Test Scripts
```go
// In replicate-exec-bug/main.go
// Flags not properly preserved during exec
os.Exec(os.Args[0], os.Args[1:], os.Environ())
```

## Fix Approaches

### Implemented Solutions

#### 1. Quick-Test Specific Endpoint
**File:** `server/server.go:1073-1119`

```go
// handleQuickTestExecRestart handles instant exec restart for quick-test mode.
// Unlike the regular exec-restart, this does not wait for graceful shutdown.
func handleQuickTestExecRestart(w http.ResponseWriter, r *http.Request) {
    // Immediate restart without graceful shutdown wait
    // Uses syscall.Exec for process replacement
}
```

#### 2. Health Check Pause Mechanism
**File:** `run/daemon/health.go:82-131`

```go
// Pauses health checks after exec-restart to allow server stabilization
type healthPauseState struct {
    paused    bool
    pauseTime time.Time
}

func pauseHealthChecks() {
    state.paused = true
    state.pauseTime = time.Now()
}
```

#### 3. Port Conflict Detection with Retry
**File:** `run/daemon/daemon.go:150-176`

```go
// Checks if port is in use with multiple retries
func waitForPortRelease(port int, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if !isPortInUse(port) {
            return nil // Port released
        }
        time.Sleep(100 * time.Millisecond)
    }
    return fmt.Errorf("port %d still in use after %v", port, timeout)
}
```

### Recommended Additional Fixes

1. **Increase Health Check Pause Duration**
   ```go
   // In health.go
   HealthCheckPauseDelay = 2 * time.Minute // Increased from 1 minute
   ```

2. **Add Pre-Restart Health Check**
   ```go
   // Before exec-restart, verify server is healthy
   if err := verifyServerHealth(); err != nil {
       return fmt.Errorf("server not healthy, aborting restart: %w", err)
   }
   ```

3. **Implement Exponential Backoff for Port Binding**
   ```go
   // Retry port binding with exponential backoff
   backoff := 100 * time.Millisecond
   maxBackoff := 5 * time.Second
   for {
       if err := bindPort(port); err == nil {
           return nil
       }
       if backoff > maxBackoff {
           return fmt.Errorf("failed to bind port after retries")
       }
       time.Sleep(backoff)
       backoff *= 2
   }
   ```

## Testing

### Test Scripts Location
- `/root/mobile-coding-connector/script/replicate-exec-bug/`
- `/root/mobile-coding-connector/script/replicate-exec-bug-fixed/`
- `/root/mobile-coding-connector/script/replicate-exec-bug-proper/`

### Reproduction Steps
1. Start server in quick-test mode
2. Trigger exec-restart via `/api/quick-test/exec-restart`
3. Monitor port binding and health check status
4. Verify process replacement occurs correctly

## Status

**Partially Fixed** - Quick-test specific endpoint implemented. General exec-restart still has race conditions that need addressing.

**Priority:** High
**Estimated Fix Time:** 2-3 days for complete resolution

---

**Related Issues:**
- Port binding race condition
- Health check timing issues
- Process orphaning during restart

**Related PRs:**
- (None yet - pending implementation)
