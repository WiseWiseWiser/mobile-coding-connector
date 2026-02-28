package common_opencode

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

// StartWebProcess starts `opencode web --port <port>` with shared process wiring.
func StartWebProcess(port int, opts *tool_exec.Options, stopChan <-chan struct{}) (*exec.Cmd, error) {
	if opts == nil {
		opts = &tool_exec.Options{}
	}

	cmdWrapper, err := tool_exec.New("opencode", []string{"web", "--port", fmt.Sprintf("%d", port)}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create opencode command: %w", err)
	}

	cmd := cmdWrapper.Cmd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		select {
		case <-stopChan:
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		case <-WaitDone(cmd):
		}
	}()

	return cmd, nil
}
