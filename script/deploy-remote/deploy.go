package main

import (
	"fmt"
)

const binaryName = "ai-critic-server-linux-amd64"

// Runner abstracts command execution, enabling testing with a mock.
type Runner interface {
	Run(args ...string) error
}

func runDeploy(runner Runner) error {
	steps := []struct {
		desc string
		args []string
	}{
		{
			desc: "Check auth",
			args: []string{"go", "run", "./cmd/remote-agent", "auth", "status"},
		},
		{
			desc: "Build binary",
			args: []string{"go", "run", "./script/bundle/for-linux/"},
		},
		{
			desc: "Upload binary",
			args: []string{"go", "run", "./cmd/remote-agent", "server", "upload-next", binaryName},
		},
		{
			desc: "Restart server",
			args: []string{"go", "run", "./cmd/remote-agent", "server", "restart"},
		},
	}

	for _, step := range steps {
		fmt.Printf("=== %s ===\n", step.desc)
		if err := runner.Run(step.args...); err != nil {
			return fmt.Errorf("%s: %w", step.desc, err)
		}
		fmt.Println()
	}

	fmt.Println("Deploy complete.")
	return nil
}
