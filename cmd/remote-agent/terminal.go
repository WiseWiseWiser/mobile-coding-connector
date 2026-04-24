package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const terminalHelp = `Usage: remote-agent terminal <subcommand> [args...]

Manage persistent remote terminal sessions.

Subcommands:
  list
      List terminal sessions.

  new [--name NAME] [cwd]
      Create a new terminal session on the remote server and attach to it.

  close <id-or-name>
      Remove a terminal session from the remote server.

  attach <id-or-name>
      Attach this terminal to an existing remote terminal session.
`

const terminalNewHelp = `Usage: remote-agent terminal new [--name NAME] [cwd]

Create a new terminal session on the remote server and attach to it.

Options:
  --name NAME          Session name shown in the frontend and CLI.
  -h, --help           Show this help message.

Arguments:
  cwd                  Optional working directory on the remote server.
`

func runTerminal(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(terminalHelp)
		return nil
	}

	switch args[0] {
	case "list":
		return runTerminalList(resolve, args[1:])
	case "new":
		return runTerminalNew(resolve, args[1:])
	case "close", "delete", "remove":
		return runTerminalClose(resolve, args[1:])
	case "attach":
		return runTerminalAttach(resolve, args[1:])
	case "-h", "--help":
		fmt.Print(terminalHelp)
		return nil
	default:
		return fmt.Errorf("unknown terminal subcommand: %s", args[0])
	}
}

func runTerminalList(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(terminalHelp)
			return nil
		}
		return fmt.Errorf("terminal list takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	sessions, err := cli.ListTerminalSessions()
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		fmt.Println("No terminal sessions found.")
		return nil
	}

	fmt.Printf("%-16s  %-8s  %-9s  %-19s  %s\n", "ID", "STATUS", "CONNECTED", "CREATED", "NAME")
	for _, session := range sessions {
		fmt.Printf("%-16s  %-8s  %-9s  %-19s  %s\n",
			session.ID,
			displayOrDash(session.Status),
			boolWord(session.Connected),
			formatAgentTime(session.CreatedAt),
			displayOrDash(session.Name),
		)
		if session.Cwd != "" {
			fmt.Printf("  cwd: %s\n", session.Cwd)
		}
	}
	return nil
}

func runTerminalNew(resolve func() (*client.Client, error), args []string) error {
	var name string
	args, err := flags.
		String("--name", &name).
		Help("-h,--help", terminalNewHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return fmt.Errorf("terminal new takes at most 1 positional argument [cwd], got %v", args)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("remote-agent terminal new requires an interactive terminal on stdin/stdout")
	}

	return runTerminalSession(resolve, terminalConnectOptions{
		name: firstArgOr(name, "Terminal"),
		cwd:  firstArg(args),
	})
}

func runTerminalClose(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Print("Usage: remote-agent terminal close <id-or-name>\n")
			return nil
		}
		return fmt.Errorf("terminal close requires exactly 1 argument <id-or-name>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	session, err := resolveTerminalTarget(cli, args[0])
	if err != nil {
		return err
	}
	if err := cli.DeleteTerminalSession(session.ID); err != nil {
		return err
	}

	fmt.Printf("Closed terminal session %s (%s)\n", session.ID, displayOrDash(session.Name))
	return nil
}

func runTerminalAttach(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Print("Usage: remote-agent terminal attach <id-or-name>\n")
			return nil
		}
		return fmt.Errorf("terminal attach requires exactly 1 argument <id-or-name>")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("remote-agent terminal attach requires an interactive terminal on stdin/stdout")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	session, err := resolveTerminalTarget(cli, args[0])
	if err != nil {
		return err
	}

	return runTerminalSession(resolve, terminalConnectOptions{
		sessionID:      session.ID,
		attachSnapshot: true,
	})
}

func resolveTerminalTarget(cli *client.Client, idOrName string) (*client.TerminalSession, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return nil, fmt.Errorf("terminal target cannot be empty")
	}

	sessions, err := cli.ListTerminalSessions()
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		if session.ID == idOrName {
			session := session
			return &session, nil
		}
	}

	var matches []client.TerminalSession
	for _, session := range sessions {
		if session.Name == idOrName {
			matches = append(matches, session)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no terminal session found for %q", idOrName)
	case 1:
		return &matches[0], nil
	default:
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, match.ID)
		}
		return nil, fmt.Errorf("terminal name %q is ambiguous; matching IDs: %s", idOrName, strings.Join(ids, ", "))
	}
}

func boolWord(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
