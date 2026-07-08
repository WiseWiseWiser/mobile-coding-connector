package daemon

import (
	"io"
	"os"

	"github.com/xhd2015/ai-critic/server/config"
)

// openManagedServerLog opens the managed server log for child stdout/stderr.
// Output is never teed to the keep-alive console so a Setpgid child cannot
// draw SIGTTOU by writing to a shared controlling terminal.
func openManagedServerLog() (io.Writer, func(), error) {
	f, err := os.OpenFile(config.ServerLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { _ = f.Close() }, nil
}