// Package gitrunner provides centralized git command execution with
// proper environment setup to suppress interactive prompts.
// This is essential for background server operations where no terminal is available.
package gitrunner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SSHKeyConfig holds SSH key configuration for git operations
type SSHKeyConfig struct {
	KeyPath string // Path to SSH private key file
}

// Command represents a git command to be executed
type Command struct {
	args      []string
	dir       string
	sshConfig *SSHKeyConfig
	env       map[string]string
	noPrompts bool // defaults to true
}

// NewCommand creates a new git command with the given arguments.
// By default, interactive prompts are suppressed.
func NewCommand(args ...string) *Command {
	return &Command{
		args:      args,
		env:       make(map[string]string),
		noPrompts: true,
	}
}

// Dir sets the working directory for the command
func (c *Command) Dir(dir string) *Command {
	c.dir = dir
	return c
}

// WithSSHKey configures the command to use an SSH key for authentication
func (c *Command) WithSSHKey(keyPath string) *Command {
	c.sshConfig = &SSHKeyConfig{KeyPath: keyPath}
	return c
}

// WithSSHConfig configures the command with a full SSH key configuration
func (c *Command) WithSSHConfig(cfg *SSHKeyConfig) *Command {
	c.sshConfig = cfg
	return c
}

// WithEnv adds an environment variable to the command
func (c *Command) WithEnv(key, value string) *Command {
	c.env[key] = value
	return c
}

// AllowPrompts allows interactive prompts (not recommended for background operations)
func (c *Command) AllowPrompts() *Command {
	c.noPrompts = false
	return c
}

// Build creates the exec.Cmd with all proper environment variables set
func (c *Command) Build() *exec.Cmd {
	cmd := exec.Command("git", c.args...)
	if c.dir != "" {
		cmd.Dir = c.dir
	}

	// Start with current environment
	env := os.Environ()

	// Add custom environment variables
	for k, v := range c.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Configure SSH if needed
	if c.sshConfig != nil && c.sshConfig.KeyPath != "" {
		// Build SSH command with key and options to prevent interactive prompts
		sshCmd := fmt.Sprintf(
			"ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o BatchMode=yes",
			c.sshConfig.KeyPath,
		)
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	}

	// Always suppress git prompts in background mode
	if c.noPrompts {
		env = append(env, "GIT_TERMINAL_PROMPT=0")
		// Prevent SSH from asking for passwords/passphrases interactively
		env = append(env, "SSH_ASKPASS_REQUIRE=never")
	}

	cmd.Env = env
	return cmd
}

// Run executes the command and returns the combined output
func (c *Command) Run() ([]byte, error) {
	cmd := c.Build()
	return cmd.CombinedOutput()
}

// Output executes the command and returns stdout only
func (c *Command) Output() ([]byte, error) {
	cmd := c.Build()
	return cmd.Output()
}

// RunSilent executes the command discarding output (useful for checks)
func (c *Command) RunSilent() error {
	cmd := c.Build()
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// Exec returns the exec.Cmd for custom execution (e.g., streaming)
func (c *Command) Exec() *exec.Cmd {
	return c.Build()
}

// Helper functions for common git operations

// Clone runs git clone with the given URL and target directory
func Clone(repoURL, targetDir string, sshKeyPath ...string) *Command {
	cmd := NewCommand("clone", "--progress", repoURL, targetDir)
	if len(sshKeyPath) > 0 && sshKeyPath[0] != "" {
		cmd.WithSSHKey(sshKeyPath[0])
	}
	return cmd
}

// Fetch runs git fetch
func Fetch(sshKeyPath ...string) *Command {
	cmd := NewCommand("fetch", "--progress")
	if len(sshKeyPath) > 0 && sshKeyPath[0] != "" {
		cmd.WithSSHKey(sshKeyPath[0])
	}
	return cmd
}

// Pull runs git pull
func Pull(sshKeyPath ...string) *Command {
	cmd := NewCommand("pull", "--progress")
	if len(sshKeyPath) > 0 && sshKeyPath[0] != "" {
		cmd.WithSSHKey(sshKeyPath[0])
	}
	return cmd
}

// PullFFOnly runs git pull --ff-only
func PullFFOnly(sshKeyPath ...string) *Command {
	cmd := NewCommand("pull", "--ff-only")
	if len(sshKeyPath) > 0 && sshKeyPath[0] != "" {
		cmd.WithSSHKey(sshKeyPath[0])
	}
	return cmd
}

// Push runs git push with explicit branch specification to avoid "no upstream branch" errors
// Usage: Push(branch, sshKeyPath) or Push(branch) - sshKeyPath is optional
func Push(branch string, sshKeyPath ...string) *Command {
	// Use origin HEAD:<branch> format to push current branch to remote without requiring upstream
	cmd := NewCommand("push", "origin", fmt.Sprintf("HEAD:%s", branch), "--progress")
	if len(sshKeyPath) > 0 && sshKeyPath[0] != "" {
		cmd.WithSSHKey(sshKeyPath[0])
	}
	return cmd
}

// Add runs git add with the given paths
func Add(paths ...string) *Command {
	args := append([]string{"add"}, paths...)
	return NewCommand(args...)
}

// Reset runs git reset HEAD with the given paths
func Reset(paths ...string) *Command {
	args := append([]string{"reset", "HEAD"}, paths...)
	return NewCommand(args...)
}

// Commit runs git commit with the given message
func Commit(message string) *Command {
	return NewCommand("commit", "-m", message)
}

// Diff runs git diff
func Diff(args ...string) *Command {
	return NewCommand(append([]string{"diff"}, args...)...)
}

// DiffCached runs git diff --cached
func DiffCached() *Command {
	return NewCommand("diff", "--cached")
}

// Status runs git status
func Status(args ...string) *Command {
	return NewCommand(append([]string{"status"}, args...)...)
}

// Branch runs git branch
func Branch(args ...string) *Command {
	return NewCommand(append([]string{"branch"}, args...)...)
}

// RevParse runs git rev-parse
func RevParse(args ...string) *Command {
	return NewCommand(append([]string{"rev-parse"}, args...)...)
}

// ForEachRef runs git for-each-ref
func ForEachRef(args ...string) *Command {
	return NewCommand(append([]string{"for-each-ref"}, args...)...)
}

// LsFiles runs git ls-files
func LsFiles(args ...string) *Command {
	return NewCommand(append([]string{"ls-files"}, args...)...)
}

// Show runs git show
func Show(args ...string) *Command {
	return NewCommand(append([]string{"show"}, args...)...)
}

// Config runs git config
func Config(key, value string) *Command {
	return NewCommand("config", key, value)
}

// IsRepo checks if the given directory is a git repository
func IsRepo(dir string) bool {
	cmd := NewCommand("rev-parse", "--git-dir").Dir(dir)
	return cmd.RunSilent() == nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(dir string) (string, error) {
	cmd := NewCommand("branch", "--show-current").Dir(dir)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
