// Package tool_exec provides a standardized way to execute external commands
// with proper environment setup, including extra PATH resolution.
//
// This is the preferred way to execute agent commands from the server,
// as it ensures consistent PATH handling and respects user configuration.
//
// For user-configured binary paths, use the CustomPath option to specify
// the exact path to the binary, which takes priority over PATH resolution.
package tool_exec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// Command represents a prepared command with proper environment setup
type Command struct {
	*exec.Cmd
}

// Options contains optional configuration for command execution
type Options struct {
	// CustomPath, if provided, will be used as the binary path directly.
	// This takes priority over PATH resolution and is typically used for
	// user-configured binary paths.
	CustomPath string
	// Env allows specifying additional environment variables
	Env map[string]string
	// Dir sets the working directory for the command
	Dir string
}

// New creates a new Command for the given binary name and arguments.
// It automatically:
// 1. Uses CustomPath if provided (for user-configured binary paths)
// 2. Otherwise resolves the binary using tool_resolve (including extra paths)
// 3. Sets up the environment with extended PATH
//
// Example:
//
//	// With user-configured path:
//	cmd, err := tool_exec.New("opencode", []string{"web", "start"}, &tool_exec.Options{
//	    CustomPath: "/custom/path/to/opencode",
//	})
//
//	// Or with automatic PATH resolution:
//	cmd, err := tool_exec.New("opencode", []string{"web", "start"}, nil)
func New(binary string, args []string, opts *Options) (*Command, error) {
	if opts == nil {
		opts = &Options{}
	}

	// Determine the actual binary path
	binaryPath, err := resolveBinaryPath(binary, opts.CustomPath)
	if err != nil {
		return nil, err
	}

	// Create the base command
	cmd := exec.Command(binaryPath, args...)

	// Setup environment with extra paths
	cmd.Env = setupEnvironment(os.Environ(), opts.Env)

	// Set working directory if specified
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	return &Command{Cmd: cmd}, nil
}

// resolveBinaryPath determines the binary path to use.
// Priority: customPath > tool_resolve search
func resolveBinaryPath(binary, customPath string) (string, error) {
	// If custom path is provided, verify and use it
	if customPath != "" {
		// If customPath is relative, resolve it
		if !filepath.IsAbs(customPath) {
			absPath, err := filepath.Abs(customPath)
			if err == nil {
				customPath = absPath
			}
		}

		// Verify the custom path exists and is executable
		if info, err := os.Stat(customPath); err == nil && !info.IsDir() {
			if info.Mode()&0111 != 0 {
				return customPath, nil
			}
		}
		// Custom path is invalid, fall through to search
	}

	// If binary is already a full path, use it directly
	if filepath.IsAbs(binary) {
		return binary, nil
	}

	// Use tool_resolve to find the binary in PATH + extra paths
	return tool_resolve.LookPath(binary)
}

// setupEnvironment prepares the environment with extra PATHs and custom vars
func setupEnvironment(baseEnv []string, extraVars map[string]string) []string {
	// Start with current environment
	env := make([]string, len(baseEnv))
	copy(env, baseEnv)

	// Append extra paths to PATH
	env = tool_resolve.AppendExtraPaths(env)

	// Add or override with custom environment variables
	if len(extraVars) > 0 {
		env = setEnvVars(env, extraVars)
	}

	return env
}

// setEnvVars sets/overrides environment variables
func setEnvVars(env []string, vars map[string]string) []string {
	// Build a map for easy lookup
	envMap := make(map[string]string)
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				key := e[:i]
				value := e[i+1:]
				envMap[key] = value
				break
			}
		}
	}

	// Override with new vars
	for k, v := range vars {
		envMap[k] = v
	}

	// Convert back to slice
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}

// MustNew is like New but panics if the binary cannot be resolved.
// Useful for commands that are expected to always succeed.
func MustNew(binary string, args []string, opts *Options) *Command {
	cmd, err := New(binary, args, opts)
	if err != nil {
		panic(fmt.Sprintf("tool_exec: failed to create command for %s: %v", binary, err))
	}
	return cmd
}

// IsAvailable checks if a binary is available.
// If customPath is provided, it checks that path; otherwise searches PATH + extra paths.
func IsAvailable(binary, customPath string) bool {
	_, err := resolveBinaryPath(binary, customPath)
	return err == nil
}
