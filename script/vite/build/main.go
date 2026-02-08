package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

var help = `
Usage: go run ./script/vite/build [options]

Builds the frontend project using Vite.

Options:
  --outdir DIR    Output directory for the build (defaults to ai-critic-react/dist)
  -h, --help      Show this help message
`

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	var outDir string
	_, err := flags.
		String("--outdir", &outDir).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	// Default output directory
	if outDir == "" {
		outDir = "ai-critic-react/dist"
	}

	// Build the frontend using Vite
	fmt.Println("Building frontend with Vite...")

	// Ensure node_modules exists
	if err := lib.EnsureNodeModules("ai-critic-react"); err != nil {
		return err
	}

	// Run npm run build in the ai-critic-react directory
	buildArgs := []string{"run", "build"}
	if outDir != "ai-critic-react/dist" {
		// If custom outdir, pass it to vite
		buildArgs = append(buildArgs, "--", "--outDir", outDir)
	}

	// Use bash to ensure nvm is loaded if available
	shellCmd := lib.WithNodejs20(fmt.Sprintf("npm %s", strings.Join(buildArgs, " ")))
	cmd := exec.Command("bash", "-c", shellCmd)
	cmd.Dir = "ai-critic-react"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build frontend: %v", err)
	}

	fmt.Printf("Frontend build complete. Output: %s\n", outDir)
	return nil
}
