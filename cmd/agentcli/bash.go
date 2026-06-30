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

const bashHelp = `Usage: remote-agent bash [--name <name>] [cwd]

Start an interactive shell on the remote server using the same terminal
WebSocket API as the frontend terminal page. Disconnecting this client does
not close the server-side terminal session.

Arguments:
  cwd                  Optional working directory on the remote server.

Options:
  --name NAME          Session name shown to the server. Defaults to "Terminal".
  -h, --help           Show this help message.

Examples:
  remote-agent bash
  remote-agent bash ~/work/repo
  remote-agent bash --name Debug /tmp
`

func runBash(resolve func() (*client.Client, error), args []string) error {
	var name string
	args, err := flags.
		String("--name", &name).
		Help("-h,--help", bashHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return fmt.Errorf("bash takes at most 1 positional argument [cwd], got %d", len(args))
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("remote-agent bash requires an interactive terminal on stdin/stdout")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	c := ptyClientFrom(cli)
	_, err = ptyclient.Attach(c, ptyclient.ConnectOptions{
		Name: firstArgOr(name, "Terminal"),
		Cwd:  firstArg(args),
		Wait: true,
	})
	return err
}

func ptyClientFrom(cli *client.Client) *ptyclient.Client {
	c := ptyclient.NewClient(cli.Server)
	c.AuthToken = cli.Token
	return c
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func firstArgOr(v string, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}