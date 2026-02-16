package lib

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func CheckPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func GetPidOnPort(port int) (int, error) {
	cmd := exec.Command("lsof", "-t", "-i", fmt.Sprintf(":%d", port))
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("no process found on port %d", port)
	}
	pidStr := string(output)
	pidStr = strings.TrimSpace(pidStr)
	if pidStr == "" {
		return 0, fmt.Errorf("no process found on port %d", port)
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, err
	}
	return pid, nil
}
