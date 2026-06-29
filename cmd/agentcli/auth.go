package agentcli

import (
	"fmt"

	"github.com/xhd2015/ai-critic/client"
)

const authHelp = `Usage: remote-agent auth <subcommand> [args...]

Auth utilities for checking server connectivity and token validity.

Subcommands:
  status
      Check authentication status against the configured server.
      Verifies the server is reachable and the token is valid.
`

const authStatusHelp = `Usage: remote-agent auth status

Check authentication status against the configured server.
Uses the default server and token from saved config, or --server/--token flags.

Exit code 0 means authenticated; non-zero means unauthenticated or error.
`

func runAuth(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(authHelp)
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "status":
		return runAuthStatus(resolve, rest)
	case "-h", "--help":
		fmt.Print(authHelp)
		return nil
	default:
		return fmt.Errorf("unknown auth subcommand: %s", sub)
	}
}

func runAuthStatus(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(authStatusHelp)
			return nil
		}
		return fmt.Errorf("auth status takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	fmt.Printf("Server: %s\n", cli.Server)

	result, err := cli.AuthStatus()
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	if result.OK {
		fmt.Println("Auth: OK")
		return nil
	}

	if !result.Initialized {
		fmt.Println("Auth: not_initialized (server has no credentials set up)")
		return fmt.Errorf("server not initialized")
	}

	fmt.Println("Auth: unauthorized (check your token)")
	return fmt.Errorf("unauthorized")
}
