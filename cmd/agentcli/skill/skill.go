package skill

import (
	"embed"

	"github.com/xhd2015/skills/skillcmd"
)

//go:embed SKILL.md
var skillRoot string

// Nested topics: path "upload" → upload/TOPIC.md (Shape 3).
//
//go:embed config
//go:embed exec
//go:embed upload
//go:embed service
//go:embed cron
//go:embed seal
//go:embed git
//go:embed server
//go:embed request
//go:embed proxy
//go:embed install
var skillTree embed.FS

const skillName = "remote-agent"

var skillHost = &skillcmd.SingleSkill{
	Name:        skillName,
	RootContent: skillRoot,
	TreeFS:      skillTree,
	Usage:       "remote-agent skill --install",
}

// Handle runs remote-agent skill (--show / --install / --list).
// Legacy word subcommands show|install|list|topics are accepted as aliases.
func Handle(args []string) error {
	return skillHost.Handle(normalizeSkillArgs(args))
}

func normalizeSkillArgs(args []string) []string {
	if len(args) == 0 {
		return []string{"--help"}
	}
	switch args[0] {
	case "show":
		return append([]string{"--show"}, args[1:]...)
	case "install":
		return append([]string{"--install"}, args[1:]...)
	case "list", "topics":
		return append([]string{"--list"}, args[1:]...)
	default:
		return args
	}
}
