package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	"golang.org/x/term"

	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const execHelp = `Usage: remote-agent exec <BINARY> [ARGS...]

Run a subprocess on the server. In non-interactive mode, stdout and stderr
are streamed back to this machine as they are produced; the client's exit
code mirrors the remote exit code.

When stdin/stdout are attached to an interactive terminal, 'exec' switches
to a PTY-backed mode so the remote process can receive live user input.

Every argument after 'exec' is forwarded verbatim to the remote process,
so there is no need for '--' or client-side flag parsing.

Examples:
  remote-agent exec ls -la /tmp
  remote-agent exec sh -c 'echo hi; sleep 1'
  remote-agent exec python3
`

// runExec is the client-side implementation of 'remote-agent exec'.
//
// By design, this subcommand does NOT use a flag parser: every argument
// after 'exec' is passed verbatim to the remote binary, so users can invoke
// commands with flags of their own (e.g. 'remote-agent exec ls -la') without
// needing '--'. The only recognized client-side token is '--help' / '-h' as
// the first argument, matching common CLI conventions.
func runExec(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("exec requires <BINARY> [ARGS...]; see 'remote-agent exec --help'")
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Print(execHelp)
		return nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		return runExecInteractive(resolve, args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	exitCode, err := cli.Exec(client.ExecRequest{Argv: args}, func(ev client.ExecEvent) {
		switch ev.Type {
		case "stdout":
			os.Stdout.WriteString(ev.Data)
		case "stderr":
			os.Stderr.WriteString(ev.Data)
		}
	})
	if err != nil {
		return err
	}

	if exitCode != 0 {
		// Exit with the same code the remote process produced, so
		// scripts wrapping 'remote-agent exec' behave correctly.
		os.Exit(normalizeExitCode(exitCode))
	}
	return nil
}

type execInteractiveServerMessage struct {
	Type    string `json:"type"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func runExecInteractive(resolve func() (*client.Client, error), args []string) error {
	cli, err := resolve()
	if err != nil {
		return err
	}

	wsURL, err := execWebSocketURL(cli)
	if err != nil {
		return err
	}

	header := http.Header{}
	if cli.Token != "" {
		header.Set("Authorization", "Bearer "+cli.Token)
	}

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return terminalDialError(err, resp)
	}
	defer conn.Close()

	writer := &wsWriter{conn: conn}
	if err := writer.writeJSON(client.ExecRequest{Argv: args}); err != nil {
		return err
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("enable raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	if err := sendTerminalResize(writer); err != nil {
		return err
	}

	sigWinch := make(chan os.Signal, 1)
	signal.Notify(sigWinch, syscall.SIGWINCH)
	defer signal.Stop(sigWinch)
	go func() {
		for range sigWinch {
			_ = sendTerminalResize(writer)
		}
	}()

	readerCh := make(chan execReadResult, 1)
	go func() {
		code, err := readExecInteractiveOutput(conn)
		readerCh <- execReadResult{exitCode: code, err: err}
	}()

	stdinErrCh := make(chan error, 1)
	go func() {
		stdinErrCh <- forwardTerminalInput(writer)
	}()

	for {
		select {
		case res := <-readerCh:
			if res.err != nil {
				return res.err
			}
			if res.exitCode != 0 {
				os.Exit(normalizeExitCode(res.exitCode))
			}
			return nil
		case err := <-stdinErrCh:
			if err == nil || err == io.EOF {
				continue
			}
			_ = conn.Close()
			res := <-readerCh
			if res.err != nil {
				return res.err
			}
			return err
		}
	}
}

type execReadResult struct {
	exitCode int
	err      error
}

func execWebSocketURL(cli *client.Client) (string, error) {
	base, err := url.Parse(cli.Server)
	if err != nil {
		return "", fmt.Errorf("invalid server url %q: %w", cli.Server, err)
	}
	switch base.Scheme {
	case "http":
		base.Scheme = "ws"
	case "https":
		base.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported server scheme %q", base.Scheme)
	}
	base.Path = "/api/exec/ws"
	base.RawQuery = ""
	return base.String(), nil
}

func readExecInteractiveOutput(conn *websocket.Conn) (int, error) {
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return 0, normalizeTerminalReadError(err)
		}
		switch msgType {
		case websocket.BinaryMessage:
			if _, err := os.Stdout.Write(data); err != nil {
				return 0, err
			}
		case websocket.TextMessage:
			var msg execInteractiveServerMessage
			if err := json.Unmarshal(data, &msg); err == nil && msg.Type != "" {
				switch msg.Type {
				case "exit":
					return msg.Code, nil
				case "error":
					if msg.Message == "" {
						return 0, fmt.Errorf("remote exec error")
					}
					return 0, fmt.Errorf("%s", msg.Message)
				default:
					if msg.Message != "" {
						if _, err := os.Stdout.WriteString(msg.Message); err != nil {
							return 0, err
						}
					}
					continue
				}
			}
			if _, err := os.Stdout.Write(data); err != nil {
				return 0, err
			}
		}
	}
}

// normalizeExitCode clamps an arbitrary integer into the 1..255 range that
// os.Exit accepts portably. -1 (our "killed by signal" sentinel) becomes 255.
func normalizeExitCode(code int) int {
	if code <= 0 {
		return 255
	}
	if code > 255 {
		return 255
	}
	return code
}
