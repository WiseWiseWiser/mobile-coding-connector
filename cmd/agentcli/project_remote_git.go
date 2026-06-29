package agentcli

import (
	"fmt"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

func remoteGitOutput(cli *client.Client, dir string, args ...string) (string, int, error) {
	argv := append([]string{"git", "-C", dir}, args...)
	var buf strings.Builder
	code, err := cli.Exec(client.ExecRequest{Argv: argv}, func(ev client.ExecEvent) {
		switch ev.Type {
		case "stdout", "stderr":
			buf.WriteString(ev.Data)
		}
	})
	return buf.String(), code, err
}

func remoteGitMust(cli *client.Client, dir string, args ...string) (string, error) {
	out, code, err := remoteGitOutput(cli, dir, args...)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("remote git %v failed (exit %d): %s", args, code, strings.TrimSpace(out))
	}
	return out, nil
}

func remoteGitOriginURL(cli *client.Client, dir string) (string, error) {
	out, err := remoteGitMust(cli, dir, "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("read remote git origin: %w", err)
	}
	origin := strings.TrimSpace(out)
	if origin == "" {
		return "", fmt.Errorf("remote repository has no origin remote")
	}
	return origin, nil
}