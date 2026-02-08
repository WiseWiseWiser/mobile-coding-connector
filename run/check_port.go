package run

import (
	"fmt"
	"net"
	"time"

	"github.com/xhd2015/less-gen/flags"
)

var checkPortHelp = `
Usage: ai-critic check-port --port PORT

Checks if a TCP port is accessible on localhost.
Exits with code 0 if the port is reachable, 1 otherwise.

Options:
  --port PORT      Port number to check (required)
  --timeout SECS   Connection timeout in seconds (default: 2)
  -h, --help       Show this help message
`

func runCheckPort(args []string) error {
	var portFlag int
	var timeoutFlag int
	_, err := flags.
		Int("--port", &portFlag).
		Int("--timeout", &timeoutFlag).
		Help("-h,--help", checkPortHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if portFlag <= 0 {
		return fmt.Errorf("--port is required")
	}

	timeout := 2 * time.Second
	if timeoutFlag > 0 {
		timeout = time.Duration(timeoutFlag) * time.Second
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", portFlag), timeout)
	if err != nil {
		return fmt.Errorf("port %d is not accessible", portFlag)
	}
	conn.Close()
	return nil
}
