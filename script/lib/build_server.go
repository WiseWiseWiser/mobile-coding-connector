package lib

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xhd2015/xgo/support/cmd"
)

// BuildServerOptions configures a server binary build.
type BuildServerOptions struct {
	Output string // Output binary path
	GOOS   string // Target OS (empty = native)
	GOARCH string // Target architecture (empty = native)
}

// BuildServer builds the Go server binary. When GOOS/GOARCH are set,
// it cross-compiles with CGO_ENABLED=0 and clears GOFLAGS.
func BuildServer(opts BuildServerOptions) error {
	if opts.Output == "" {
		return fmt.Errorf("output path is required")
	}

	isCross := opts.GOOS != "" || opts.GOARCH != ""

	if isCross {
		return buildCross(opts)
	}
	return buildNative(opts)
}

func buildNative(opts BuildServerOptions) error {
	fmt.Printf("Building Go server -> %s\n", opts.Output)
	if err := cmd.Debug().Run("go", "build", "-o", opts.Output, "./"); err != nil {
		return fmt.Errorf("failed to build Go server: %v", err)
	}
	fmt.Printf("Server binary built: %s\n", opts.Output)
	return nil
}

func buildCross(opts BuildServerOptions) error {
	target := opts.GOOS + "/" + opts.GOARCH
	fmt.Printf("Cross-compiling Go server for %s -> %s\n", target, opts.Output)

	// Clear GOFLAGS to avoid inheriting host-specific flags like -linkmode=external
	// which conflict with CGO_ENABLED=0 cross-compilation.
	env := FilterEnv(os.Environ(), "GOFLAGS")
	if opts.GOOS != "" {
		env = append(env, "GOOS="+opts.GOOS)
	}
	if opts.GOARCH != "" {
		env = append(env, "GOARCH="+opts.GOARCH)
	}
	env = append(env, "CGO_ENABLED=0")

	buildCmd := exec.Command("go", "build", "-ldflags=", "-o", opts.Output, "./")
	buildCmd.Env = env
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("cross-compile for %s failed: %v", target, err)
	}
	fmt.Printf("Server binary built: %s\n", opts.Output)
	return nil
}

// BuildFrontend builds the frontend using Vite (npm run build in ai-critic-react).
func BuildFrontend() error {
	fmt.Println("Building frontend with Vite...")
	if err := cmd.Dir("ai-critic-react").Debug().Run("npm", "run", "build"); err != nil {
		return fmt.Errorf("failed to build frontend: %v", err)
	}
	fmt.Println("Frontend build complete.")
	return nil
}

// FilterEnv returns env with the specified keys removed.
func FilterEnv(env []string, keys ...string) []string {
	result := make([]string, 0, len(env))
	for _, e := range env {
		skip := false
		for _, key := range keys {
			if strings.HasPrefix(e, key+"=") {
				skip = true
				break
			}
		}
		if !skip {
			result = append(result, e)
		}
	}
	return result
}
