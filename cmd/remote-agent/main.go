package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const help = `Usage: remote-agent [--server URL] [--token TOKEN] <command> [args...]

A CLI that talks to the ai-critic server's HTTP API, mirroring the
behavior of the web frontend.

Global options:
  --server URL       Base URL of the server (e.g. https://host.example.com).
                     Falls back to the default domain saved by 'config'.
  --token TOKEN      Bearer token for authentication. Falls back to the
                     token of the matching/default saved domain.
  -h, --help         Show this help message

Commands:
  config
      Open a local web page to manage saved server domains and pick a
      default one. When 'upload' is invoked without --server, the
      default domain's server and token are used.

  upload <LOCAL_FILE> [REMOTE_PATH]
      Upload a local file to the server using chunked upload.
      If REMOTE_PATH is omitted, the file's basename is used.
      If REMOTE_PATH ends with '/', the basename is appended.

Examples:
  remote-agent config
  remote-agent --server https://host.example.com --token abc upload ./foo.txt /tmp/foo.txt
  remote-agent upload ./foo.txt /tmp/          # uses saved config
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var server string
	var token string

	args, err := flags.
		String("--server", &server).
		String("--token", &token).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		fmt.Print(help)
		return nil
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "config":
		return runConfig(rest)
	case "upload":
		cli, err := resolveClient(server, token)
		if err != nil {
			return err
		}
		return runUpload(cli, rest)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// resolveClient builds a client.Client using CLI flags, falling back to the
// default domain saved via 'remote-agent config'.
func resolveClient(server string, token string) (*client.Client, error) {
	cfg, _ := loadConfig()

	// When --server is not supplied, use the saved default domain's
	// server + token. --token alone can still override the saved token.
	if server == "" {
		def := cfg.DefaultDomain()
		if def == nil {
			return nil, fmt.Errorf("no server specified and no default domain configured. " +
				"Pass --server, or run 'remote-agent config' to add a domain and mark it as default.")
		}
		server = def.Server
		if token == "" {
			token = def.Token
		}
	}

	return client.New(server, token), nil
}
