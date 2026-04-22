package main

import (
	"fmt"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const serverHelp = `Usage: remote-agent server <subcommand> [args...]

Execute server-management actions exposed by the remote server UI.

Subcommands:
  build-next [--project <id>]
      Trigger the same "Build Next" action as the Manage Server page.
      Build logs are streamed back live.

  restart
      Trigger the same "Restart Server" action as the Manage Server page.
      Restart progress is streamed back live.
`

const serverBuildNextHelp = `Usage: remote-agent server build-next [--project <id>]

Trigger the remote server's /api/build/build-next action and stream its
logs back to this terminal.

Options:
  --project ID       Build the specified buildable project. If omitted,
                     the server chooses the same default project as the UI.
  -h, --help         Show this help message.
`

const serverRestartHelp = `Usage: remote-agent server restart

Trigger the remote server's /api/server/exec-restart action and stream
restart progress back to this terminal.
`

func runServer(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(serverHelp)
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "build-next":
		return runServerBuildNext(resolve, rest)
	case "restart":
		return runServerRestart(resolve, rest)
	case "-h", "--help":
		fmt.Print(serverHelp)
		return nil
	default:
		return fmt.Errorf("unknown server subcommand: %s", sub)
	}
}

func runServerBuildNext(resolve func() (*client.Client, error), args []string) error {
	var projectID string

	args, err := flags.
		String("--project", &projectID).
		Help("-h,--help", serverBuildNextHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("server build-next does not accept positional args: %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	var result *client.BuildNextResult
	result, err = cli.BuildNext(projectID, func(ev client.ServerStreamEvent) {
		if ev.Message != "" {
			fmt.Println(ev.Message)
		}
	})
	if err != nil {
		return err
	}

	if result != nil {
		fmt.Printf("Build complete: %s\n", result.BinaryPath)
		if result.ProjectName != "" || result.Version != "" {
			fmt.Printf("Project: %s  Version: %s\n", displayOrDash(result.ProjectName), displayOrDash(result.Version))
		}
	}
	return nil
}

func runServerRestart(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(serverRestartHelp)
			return nil
		}
		return fmt.Errorf("server restart takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	result, err := cli.RestartServer(func(ev client.ServerStreamEvent) {
		if ev.Message != "" {
			fmt.Println(ev.Message)
		}
	})
	if err != nil {
		return err
	}

	if result != nil && result.Binary != "" {
		fmt.Printf("Restart requested with binary: %s\n", result.Binary)
	}
	return nil
}
