package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/xgo/support/cmd"
)

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	// Step 1: Build frontend
	fmt.Println("=== Building frontend ===")

	// Check if node_modules exists, run npm install if not
	if _, err := os.Stat("ai-critic-react/node_modules"); err != nil {
		fmt.Println("node_modules not found, running npm install...")
		err := cmd.Debug().Dir("ai-critic-react").Run("npm", "install")
		if err != nil {
			return fmt.Errorf("npm install failed: %v", err)
		}
	}

	// Build frontend with npm
	err := cmd.Debug().Dir("ai-critic-react").Run("npm", "run", "build")
	if err != nil {
		return fmt.Errorf("frontend build failed: %v", err)
	}
	fmt.Println("Frontend build complete.")

	// Step 2: Build server
	fmt.Println("\n=== Building server ===")
	err = cmd.Debug().Run("go", "run", "./script/server/build")
	if err != nil {
		return fmt.Errorf("server build failed: %v", err)
	}
	fmt.Println("\nBuild complete!")

	return nil
}
