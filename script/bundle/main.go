// Command bundle builds the React frontend and compiles the Go backend
// for the host OS/arch into a single self-contained ai-critic binary.
//
// Usage (from the ai-critic module root):
//
//	go run ./script/bundle
//
// The resulting artifact is written to
// ./ai-critic-server-<goos>-<goarch> in the module root so host and
// cross-compiled bundles can coexist.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	fmt.Printf("module root: %s\n", root)

	if err := buildFrontend(); err != nil {
		return err
	}

	outputName := fmt.Sprintf("%s-%s-%s", lib.BinaryName, runtime.GOOS, runtime.GOARCH)
	if err := lib.BuildServer(lib.BuildServerOptions{
		Output: outputName,
	}); err != nil {
		return err
	}

	out := filepath.Join(root, outputName)
	fmt.Printf("\nBundle ready: %s\n", out)
	return nil
}

func buildFrontend() error {
	if err := lib.EnsureNodeModules("ai-critic-react"); err != nil {
		return err
	}
	return lib.BuildFrontend()
}
