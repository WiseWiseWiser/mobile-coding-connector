package agentcli

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/xhd2015/ai-critic/client"
	ptyclient "github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/client"
	"github.com/xhd2015/less-gen/flags"
)

const terminalHelp = `Usage: remote-agent terminal <subcommand> [args...]

Manage persistent remote terminal sessions.

Subcommands:
  list
      List terminal sessions.

  new [--name NAME] [cwd]
      Create a new terminal session on the remote server and attach to it.
      Detach without stopping the remote shell: Ctrl-].

  close <id-or-name>
      Remove a terminal session from the remote server.

  attach <id-or-name>
      Attach this terminal to an existing remote terminal session.
      Detach without stopping the remote shell: Ctrl-].
`

const terminalNewHelp = `Usage: remote-agent terminal new [--name NAME] [cwd]

Create a new terminal session on the remote server and attach to it.

Detach without stopping the remote shell: Ctrl-].

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
	c := ptyClientFrom(cli)
	sessions, err := c.List()
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
			formatAgentTime(session.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
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

	cli, err := resolve()
	if err != nil {
		return err
	}
	c := ptyClientFrom(cli)
	result, err := ptyclient.Attach(c, ptyclient.ConnectOptions{
		Name: firstArgOr(name, "Terminal"),
		Cwd:  firstArg(args),
		Wait: true,
	})
	if err != nil {
		return err
	}
	printDetached(result)
	return nil
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
	c := ptyClientFrom(cli)
	session, err := ptyclient.ResolveTarget(c, args[0])
	if err != nil {
		return err
	}
	if err := c.Delete(session.ID); err != nil {
		return err
	}

	fmt.Printf("Closed terminal session %s (%s)\n", session.ID, displayOrDash(session.Name))
	return nil
}

func runTerminalAttach(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Print("Usage: remote-agent terminal attach <id-or-name>\n\nDetach without stopping the remote shell: Ctrl-].\n")
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
	c := ptyClientFrom(cli)
	session, err := ptyclient.ResolveTarget(c, args[0])
	if err != nil {
		return err
	}
	if err := ErrIfSessionNotAttachable(*session); err != nil {
		return err
	}

	result, err := ptyclient.Attach(c, TerminalAttachConnectOptions(session.ID))
	if err != nil {
		return err
	}
	printDetached(result)
	return nil
}

func printDetached(result ptyclient.AttachResult) {
	if !result.Detached {
		return
	}
	id := strings.TrimSpace(result.SessionID)
	if id == "" {
		fmt.Println("detached (session still running)")
		return
	}
	fmt.Printf("detached (%s still running)\n", id)
}

// ErrIfSessionNotAttachable rejects sessions whose PTY child is already gone.
// Attach would otherwise open a mute raw TTY on replayed scrollback.
func ErrIfSessionNotAttachable(session ptyclient.SessionInfo) error {
	if strings.EqualFold(strings.TrimSpace(session.Status), "exited") {
		return fmt.Errorf("%s is exited", session.ID)
	}
	return nil
}

// TerminalAttachConnectOptions is the handshake `remote-agent terminal attach`
// uses for an existing session. attach_mode=attach claims roleAttacher so
// keystrokes are accepted even when another client already holds the writer.
func TerminalAttachConnectOptions(sessionID string) ptyclient.ConnectOptions {
	return ptyclient.ConnectOptions{
		SessionID:  sessionID,
		AttachMode: "attach",
		Wait:       true,
	}
}

func boolWord(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}