// Command for-linux builds the React frontend and cross-compiles the Go
// backend into a single static Linux amd64 ai-critic binary.
//
// Usage (from the ai-critic module root):
//
//	go run ./script/bundle/for-linux
//
// The resulting artifact is written to
// ./ai-critic-server-linux-amd64 in the module root.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

const outputName = "ai-critic-server-linux-amd64"

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

	if err := lib.BuildServer(lib.BuildServerOptions{
		Output: outputName,
		GOOS:   "linux",
		GOARCH: "amd64",
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
