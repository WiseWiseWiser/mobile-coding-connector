package wsproxy

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Sessions counts established TCP connections to the xray listen port. Every
// active phone session arrives as one (or more) ESTABLISHED connection from
// the local cloudflared, so the count is per-stream, not per-device; per-user
// breakdown needs an Xray stats API integration (not wired yet).
//
// Sampling parses `netstat -an`, which exists with near-identical output on
// macOS (dot separators) and Linux (colon separators).
func Sessions(port int) int {
	if port <= 0 {
		return 0
	}
	out, err := exec.Command("netstat", "-an").Output()
	if err != nil {
		return 0
	}
	suffixDot := "." + strconv.Itoa(port) + " "
	suffixColon := ":" + strconv.Itoa(port) + " "
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		state := fields[len(fields)-1]
		if state != "ESTABLISHED" && state != "ESTAB" {
			continue
		}
		local := fields[3]
		if strings.HasSuffix(local, suffixDot) || strings.HasSuffix(local, suffixColon) {
			count++
		}
	}
	return count
}

// ConnSample is one sampled established connection for the detail page.
type ConnSample struct {
	Local string `json:"local"`
	Peer  string `json:"peer"`
	Since string `json:"-"`
}

// SampleConnections returns the established connections to the xray port with
// their peer addresses (as seen by the host: cloudflared loopback conns).
func SampleConnections(port int) ([]ConnSample, int) {
	if port <= 0 {
		return nil, 0
	}
	out, err := exec.Command("netstat", "-an").Output()
	if err != nil {
		return nil, 0
	}
	suffixDot := "." + strconv.Itoa(port) + " "
	suffixColon := ":" + strconv.Itoa(port) + " "
	now := time.Now().Format(time.RFC3339)
	var samples []ConnSample
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		state := fields[len(fields)-1]
		if state != "ESTABLISHED" && state != "ESTAB" {
			continue
		}
		local, peer := fields[3], fields[4]
		if strings.HasSuffix(local, suffixDot) || strings.HasSuffix(local, suffixColon) {
			samples = append(samples, ConnSample{Local: local, Peer: peer, Since: now})
		}
	}
	return samples, len(samples)
}

// portReachable is a cheap liveness probe used by tests.
func portReachable(port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

var _ = runtime.GOOS // netstat output handled uniformly for mac/linux
var _ = fmt.Sprintf
