package lib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// WithNodejs20 wraps a shell command so that nvm is loaded and node 20 is
// activated before running the command. If nvm is not available, the command
// runs with whatever node version is in PATH.
func WithNodejs20(cmd string) string {
	return fmt.Sprintf(
		`export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" && nvm use 20 2>/dev/null || true && %s`,
		cmd,
	)
}

// EnsureNodeModules checks if node_modules exists in the given directory,
// and runs npm install if it doesn't.
func EnsureNodeModules(dir string) error {
	nodeModulesPath := filepath.Join(dir, "node_modules")
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		fmt.Printf("node_modules not found in %s, running npm install...\n", dir)
		cmd := exec.Command("npm", "install")
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm install failed: %w", err)
		}
		fmt.Println("npm install completed successfully")
	}
	return nil
}
