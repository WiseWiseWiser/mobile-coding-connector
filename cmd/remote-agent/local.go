package main

import "fmt"

const localHelp = `Usage: remote-agent local <subcommand> [args...]

Local-machine utilities that do not talk to the server.

Subcommands:
  reap [--signal] [--kill-parent] [--filter NAME]
      List defunct (zombie) processes on the local machine.
`

func runLocal(args []string) error {
	if len(args) == 0 {
		fmt.Print(localHelp)
		return nil
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "reap":
		return runLocalReap(rest)
	case "-h", "--help":
		fmt.Print(localHelp)
		return nil
	default:
		return fmt.Errorf("unknown local subcommand: %s", sub)
	}
}
