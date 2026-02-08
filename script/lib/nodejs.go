package lib

import "fmt"

// WithNodejs20 wraps a shell command so that nvm is loaded and node 20 is
// activated before running the command. If nvm is not available, the command
// runs with whatever node version is in PATH.
func WithNodejs20(cmd string) string {
	return fmt.Sprintf(
		`export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" && nvm use 20 2>/dev/null || true && %s`,
		cmd,
	)
}
