package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/xgo/support/cmd"
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

	// Run npm run build in the ai-critic-react directory
	buildArgs := []string{"run", "build"}
	if outDir != "ai-critic-react/dist" {
		// If custom outdir, pass it to vite
		buildArgs = append(buildArgs, "--", "--outDir", outDir)
	}

	err = cmd.Dir("ai-critic-react").Debug().Run("npm", buildArgs...)
	if err != nil {
		return fmt.Errorf("failed to build frontend: %v", err)
	}

	fmt.Printf("Frontend build complete. Output: %s\n", outDir)
	return nil
}
