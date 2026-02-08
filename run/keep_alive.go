package run

import (
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/terminal"
)

func runKeepAliveScript(args []string) error {
	// Reconstruct the ai-critic command: the binary path + all original
	// args except the "keep-alive-script" subcommand itself.
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	// Build the command with remaining args (those after "keep-alive-script")
	var cmdParts []string
	cmdParts = append(cmdParts, terminal.ShellQuote(binPath))
	for _, a := range args {
		cmdParts = append(cmdParts, terminal.ShellQuote(a))
	}
	serverCmd := strings.Join(cmdParts, " ")

	script := fmt.Sprintf(`#!/bin/sh
LOG_FILE="ai-critic-server.log"
RESTART_DELAY=3

while true; do
  echo "[$(date)] Starting ai-critic server..."
  if command -v tee >/dev/null 2>&1; then
    %s 2>&1 | tee -a "$LOG_FILE"
  else
    %s 2>&1
  fi
  EXIT_CODE=$?
  echo "[$(date)] ai-critic exited with code $EXIT_CODE, restarting in ${RESTART_DELAY}s..."
  sleep "$RESTART_DELAY"
done
`, serverCmd, serverCmd)

	fmt.Print(script)
	return nil
}
