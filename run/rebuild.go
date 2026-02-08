package run

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/server/terminal"
)

var rebuildHelp = `
Usage: ai-critic rebuild --repo-dir DIR [server-options...]

Rebuilds the ai-critic server binary from source and restarts it.

With --script, outputs a shell script instead of executing directly.

Options:
  --repo-dir DIR   Path to the ai-critic source repository (required)
  --script         Output a shell script instead of executing
  -h, --help       Show this help message
`

func runRebuild(args []string) error {
	var repoDir string
	var scriptFlag bool
	args, err := flags.
		String("--repo-dir", &repoDir).
		Bool("--script", &scriptFlag).
		Help("-h,--help", rebuildHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if repoDir == "" {
		return fmt.Errorf("--repo-dir is required")
	}

	// Resolve to absolute path
	repoDir, err = filepath.Abs(repoDir)
	if err != nil {
		return fmt.Errorf("resolve repo-dir: %w", err)
	}

	// Validate that repo-dir points to ai-critic source
	buildScript := filepath.Join(repoDir, "script", "server", "build", "for-linux-amd64")
	if _, err := os.Stat(buildScript); err != nil {
		return fmt.Errorf("--repo-dir %s does not appear to be an ai-critic source directory (script/server/build/for-linux-amd64 not found)", repoDir)
	}

	// Resolve the current binary path for the output target
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	binPath, err = filepath.Abs(binPath)
	if err != nil {
		return fmt.Errorf("resolve executable abs path: %w", err)
	}

	if scriptFlag {
		return outputRebuildScript(repoDir, binPath, args)
	}
	return executeRebuild(repoDir, binPath, args)
}

// executeRebuild rebuilds the binary from source and exec's it.
func executeRebuild(repoDir, binPath string, serverArgs []string) error {
	binDir := filepath.Dir(binPath)

	// Step 1: Build
	fmt.Printf("[%s] Rebuilding ai-critic from %s ...\n", timestamp(), repoDir)
	buildCmd := exec.Command("go", "run", "./script/server/build/for-linux-amd64", "-o", binPath)
	buildCmd.Dir = repoDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	fmt.Printf("[%s] Build complete: %s\n", timestamp(), binPath)

	// Ensure the binary is executable
	os.Chmod(binPath, 0755)

	// Step 2: Change to binary directory
	if err := os.Chdir(binDir); err != nil {
		return fmt.Errorf("chdir to %s: %w", binDir, err)
	}

	// Step 3: Exec the rebuilt binary (replaces current process)
	fmt.Printf("[%s] Starting ai-critic server...\n", timestamp())
	return syscall.Exec(binPath, append([]string{binPath}, serverArgs...), os.Environ())
}

// outputRebuildScript outputs a shell script for rebuilding.
func outputRebuildScript(repoDir, binPath string, serverArgs []string) error {
	binDir := filepath.Dir(binPath)

	var serverArgsStr string
	for _, a := range serverArgs {
		serverArgsStr += " " + terminal.ShellQuote(a)
	}

	script := fmt.Sprintf(`#!/bin/sh
set -e

REPO_DIR=%s
BIN_PATH=%s
BIN_DIR=%s

echo "[$(date)] Rebuilding ai-critic from $REPO_DIR ..."
(cd "$REPO_DIR" && go run ./script/server/build/for-linux-amd64 -o "$BIN_PATH")
echo "[$(date)] Build complete: $BIN_PATH"
chmod +x "$BIN_PATH"
echo "[$(date)] Starting ai-critic server..."
cd "$BIN_DIR"
exec "$BIN_PATH"%s
`, terminal.ShellQuote(repoDir), terminal.ShellQuote(binPath), terminal.ShellQuote(binDir), serverArgsStr)

	fmt.Print(script)
	return nil
}
