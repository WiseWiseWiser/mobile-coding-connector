package procmon

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// KillResult is the JSON body for a successful kill.
type KillResult struct {
	Status string `json:"status"`
	PID    int    `json:"pid"`
	Signal string `json:"signal"`
}

func parseSignal(name string) (syscall.Signal, string, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "", "TERM", "SIGTERM", "15":
		return syscall.SIGTERM, "TERM", nil
	case "KILL", "SIGKILL", "9":
		return syscall.SIGKILL, "KILL", nil
	default:
		return 0, "", fmt.Errorf("unsupported signal %q (TERM or KILL)", name)
	}
}

func inspectPID(pid int) (rawProc, error) {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pid,ppid,uid,user,state,rss,etime,time,command")
	out, err := cmd.Output()
	if err != nil {
		return rawProc{}, fmt.Errorf("process not found")
	}
	procs, err := parsePSTable(string(out))
	if err != nil || len(procs) == 0 {
		return rawProc{}, fmt.Errorf("process not found")
	}
	for _, p := range procs {
		if p.PID == pid {
			return p, nil
		}
	}
	return rawProc{}, fmt.Errorf("process not found")
}

func selfAncestorPIDs(selfPID int) map[int]struct{} {
	out := map[int]struct{}{selfPID: {}}
	pid := selfPID
	for i := 0; i < 32; i++ {
		cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "ppid=")
		b, err := cmd.Output()
		if err != nil {
			break
		}
		ppid, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil || ppid <= 0 {
			break
		}
		out[ppid] = struct{}{}
		if ppid <= 1 {
			break
		}
		pid = ppid
	}
	return out
}

// KillProcess sends signal to pid after applying protection rules.
func KillProcess(pid int, signalName string) (*KillResult, error) {
	sig, label, err := parseSignal(signalName)
	if err != nil {
		return nil, err
	}
	if pid <= 1 {
		return nil, &killError{status: 403, msg: "cannot kill init process (PID <= 1)"}
	}
	proc, err := inspectPID(pid)
	if err != nil {
		return nil, &killError{status: 404, msg: err.Error()}
	}
	reason := protectReason(proc, selfAncestorPIDs(os.Getpid()))
	if reason != "" {
		return nil, &killError{status: 403, msg: reason}
	}
	if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
		if err == syscall.ESRCH {
			return nil, &killError{status: 404, msg: "process not found"}
		}
		if err != syscall.EPERM {
			return nil, &killError{status: 500, msg: err.Error()}
		}
	}
	if err := syscall.Kill(pid, sig); err != nil {
		if err == syscall.ESRCH {
			return nil, &killError{status: 404, msg: "process not found"}
		}
		if err == syscall.EPERM {
			return nil, &killError{status: 403, msg: "permission denied"}
		}
		return nil, &killError{status: 500, msg: fmt.Sprintf("failed to kill process %d: %v", pid, err)}
	}
	return &KillResult{Status: "ok", PID: pid, Signal: label}, nil
}

type killError struct {
	status int
	msg    string
}

func (e *killError) Error() string { return e.msg }

func killStatus(err error) int {
	if ke, ok := err.(*killError); ok {
		return ke.status
	}
	return 500
}
