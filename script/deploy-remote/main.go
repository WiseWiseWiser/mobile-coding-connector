package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/xgo/support/cmd"
)

// TODO: we currently run bundle/for-linux, maybe can detect target os and build for os+arch correspondingly

const help = `Usage: deploy-remote [options]

Deploy the ai-critic server to the remote host configured in remote-agent.
Steps:
  1. Check auth  (go run ./cmd/remote-agent auth status)
  2. Build binary (go run ./script/bundle/for-linux/)
  3. Upload       (go run ./cmd/remote-agent server upload-next)
  4. Restart      (go run ./cmd/remote-agent server restart)

Options:
  --dry-run   Print steps without executing
  -h, --help  Show this help message
`

func main() {
	if err := runMain(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMain(args []string) error {
	var dryRun bool

	_, err := flags.
		Bool("--dry-run", &dryRun).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if dryRun {
		return runDeploy(&dryRunRunner{})
	}
	return runDeploy(&realRunner{})
}

type realRunner struct{}

func (r *realRunner) Run(args ...string) error {
	return cmd.Debug().Run(args[0], args[1:]...)
}

type dryRunRunner struct{}

func (r *dryRunRunner) Run(args ...string) error {
	fmt.Printf("  [DRY RUN] %v\n", args)
	return nil
}
