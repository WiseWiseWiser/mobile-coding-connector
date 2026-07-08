//go:build unix

package run

import (
	"os"
	"os/signal"
	"syscall"
)

// ignoreJobControlStop prevents SIGTSTP from freezing the server when it shares
// a controlling terminal with a remote exec / nohup parent that cannot Setsid.
func ignoreJobControlStop() {
	if !isManagedServerChild() {
		return
	}
	signal.Ignore(syscall.SIGTSTP)
}

// isManagedServerChild reports keep-alive-spawned servers (--port set, no subcommand).
func isManagedServerChild() bool {
	args := os.Args[1:]
	if len(args) == 0 {
		return false
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "keep-alive", "rebuild", "check-port":
			return false
		case "--port":
			return i+1 < len(args)
		}
	}
	return false
}