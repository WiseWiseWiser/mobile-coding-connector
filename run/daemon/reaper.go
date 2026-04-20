package daemon

import (
	"syscall"
	"time"
)

// ZombieReapInterval controls how often the daemon sweeps for defunct
// (zombie) child processes. The daemon spawns the server via
// exec.Command and also runs short-lived helpers (lsof, sh, ...). If a
// child ever exits without us calling Wait(), it would become a zombie
// under the daemon's PID. This periodic sweep reclaims any such
// orphans in a non-blocking way so 'ps -ef' stays clean.
const ZombieReapInterval = 30 * time.Second

// StartZombieReaper launches a background goroutine that periodically
// non-blocking-wait()s on ANY of the daemon's children. Reaping a
// child that a goroutine is also waiting on would be a race; to avoid
// that, we only use wait4(-1, ..., WNOHANG, ...) which consumes
// already-exited children and returns immediately otherwise. If
// another goroutine races to wait on the same child, one of them gets
// ECHILD — we treat that as "nothing to do" and return.
//
// Call once from the daemon's Run() after the HTTP server is started.
func StartZombieReaper() {
	go zombieReapLoop()
}

func zombieReapLoop() {
	ticker := time.NewTicker(ZombieReapInterval)
	defer ticker.Stop()

	for range ticker.C {
		reaped := reapZombiesOnce()
		if reaped > 0 {
			Logger("[reaper] reaped %d defunct child process(es)", reaped)
		}
	}
}

// reapZombiesOnce drains all currently-reapable child processes using
// non-blocking wait4. Returns the number of children reaped.
func reapZombiesOnce() int {
	n := 0
	for {
		var ws syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
		if err != nil {
			// ECHILD: no children at all, or all remaining children
			// are already being waited on by another goroutine. Either
			// way, nothing more to do this cycle.
			return n
		}
		if pid == 0 {
			// Children exist but none are ready to be reaped.
			return n
		}
		// Successfully reaped one.
		Logger("[reaper] reaped child pid=%d (exit=%d, signal=%v)", pid, ws.ExitStatus(), ws.Signal())
		n++
	}
}
