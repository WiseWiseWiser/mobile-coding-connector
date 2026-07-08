//go:build unix

package daemon

import "syscall"

// serverChildProcAttr configures the managed server child process group.
// When detach is true (explicit --detach or auto-detect via non-tty stdin),
// Setsid gives the child its own session without a controlling terminal.
func serverChildProcAttr(detach bool) *syscall.SysProcAttr {
	if detach {
		return &syscall.SysProcAttr{
			Setsid:  true,
			Setpgid: true,
		}
	}
	return &syscall.SysProcAttr{Setpgid: true}
}