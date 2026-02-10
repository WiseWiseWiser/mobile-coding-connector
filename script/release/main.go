package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

var binaryName = lib.BinaryName

// targets defines the cross-compilation targets for release.
var targets = []struct {
	GOOS   string
	GOARCH string
}{
	{"linux", "amd64"},
	{"linux", "arm64"},
}

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	// Step 1: Build frontend (shared across all targets)
	fmt.Println("=== Building frontend ===")
	if err := lib.BuildFrontend(); err != nil {
		return err
	}

	// Step 2: Cross-compile for each target
	for _, t := range targets {
		output := fmt.Sprintf("%s-%s-%s", binaryName, t.GOOS, t.GOARCH)
		fmt.Printf("\n=== Building %s/%s -> %s ===\n", t.GOOS, t.GOARCH, output)
		if err := lib.BuildServer(lib.BuildServerOptions{
			Output: output,
			GOOS:   t.GOOS,
			GOARCH: t.GOARCH,
		}); err != nil {
			return fmt.Errorf("build %s/%s failed: %v", t.GOOS, t.GOARCH, err)
		}
	}

	fmt.Println("\n=== Release build complete! ===")
	fmt.Println("Binaries:")
	for _, t := range targets {
		output := fmt.Sprintf("%s-%s-%s", binaryName, t.GOOS, t.GOARCH)
		fmt.Printf("  %s\n", output)
	}
	fmt.Println("\nUpload these binaries to a GitHub release.")
	return nil
}
