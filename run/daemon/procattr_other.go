//go:build !unix

package daemon

import "syscall"

func serverChildProcAttr(detach bool) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}