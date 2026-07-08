//go:build unix

package daemon

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// stdinIsTerminal reports whether keep-alive stdin is an interactive terminal.
func stdinIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// resolveEffectiveDetach enables session detach when --detach is set or when
// stdin is not a terminal (e.g. nohup redirects stdin to /dev/null).
func resolveEffectiveDetach(explicitDetach bool) bool {
	return explicitDetach || !stdinIsTerminal()
}

// ignoreTerminalHangup ignores SIGHUP so a nohup-launched daemon survives
// terminal hangup while the new session is being established.
func ignoreTerminalHangup() {
	signal.Ignore(syscall.SIGHUP)
}

// tryDetachSession calls setsid(2). Containers that block session creation
// (EPERM) must not pass Setsid to the managed server child or exec.Start fails.
func tryDetachSession() bool {
	_, err := syscall.Setsid()
	return err == nil
}