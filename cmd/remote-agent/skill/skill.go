package skill

import (
	_ "embed"
	"fmt"

	"github.com/xhd2015/less-gen/flags"
)

//go:embed SKILL.md
var skillTemplate string

const help = `Usage: remote-agent skill <command> [args...]

Manage the embedded remote-agent skill definition.

Commands:
  install [<dir>]      Install SKILL.md to a directory, or use --cursor/--codex

Examples:
  remote-agent skill install --codex
  remote-agent skill install --cursor
  remote-agent skill install ./tmp/remote-agent-skill
`

func Handle(args []string) error {
	args, err := flags.
		Help("-h,--help", help).
		StopOnFirstArg().
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		fmt.Print(help)
		return nil
	}

	switch args[0] {
	case "install", "create-skill":
		return handleInstall(args[1:])
	default:
		return fmt.Errorf("unknown skill command: %s", args[0])
	}
}

func handleInstall(args []string) error {
	return HandleInstall(InstallOptions{
		CursorDirName: "remote-agent",
		SkillContent:  skillTemplate,
	}, args)
}
