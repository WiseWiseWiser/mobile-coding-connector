package agentcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

const projectBindLocalHelp = `Usage: remote-agent project bind-local <remote-dir> <local-path>

Save a local git repository path for a remote project after verifying both
repos share the same origin remote.
`

func runProjectBindLocal(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 2 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Print(projectBindLocalHelp)
			return nil
		}
		return fmt.Errorf("project bind-local requires <remote-dir> <local-path>")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	project, err := resolveProjectTarget(cli, args[0])
	if err != nil {
		return err
	}

	localPath, err := filepath.Abs(strings.TrimSpace(args[1]))
	if err != nil {
		return fmt.Errorf("resolve local path: %w", err)
	}
	if _, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("local path %s: %w", localPath, err)
	}

	localOrigin, err := localGitOriginURL(localPath)
	if err != nil {
		return err
	}

	remoteOrigin, err := remoteGitOriginURL(cli, project.Dir)
	if err != nil {
		return err
	}

	if !sameGitOrigin(localOrigin, remoteOrigin) {
		return fmt.Errorf("git origin mismatch: local %q vs remote %q", localOrigin, remoteOrigin)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &agentConfig{}
	}
	remoteDir := filepath.Clean(project.Dir)
	if err := upsertProjectBinding(cfg, cli.Server, remoteDir, localPath); err != nil {
		return err
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Bound project %s (%s) to local path %s\n", project.Name, project.ID, localPath)
	return nil
}