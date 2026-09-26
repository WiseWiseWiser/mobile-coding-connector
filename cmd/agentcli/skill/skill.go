package skill

import (
	"embed"

	"github.com/xhd2015/skills/skillcmd"
)

// skillRoot is the root SKILL.md served by `remote-agent skill --show`.
//
// SSOT: this directory. After edits, mirror SKILL.md and topics to
// $AI/skills/remote-agent (downstream copy). TestSkillIndexMatchesTree
// guards index↔tree consistency.
//go:embed SKILL.md
var skillRoot string

// Nested topics: path "upload" → upload/TOPIC.md (Shape 3).
// SSOT: this directory; mirror <topic>/TOPIC.md to $AI/skills/remote-agent
// after edits.
//
//go:embed config
//go:embed alias
//go:embed exec
//go:embed upload
//go:embed edit
//go:embed service
//go:embed cron
//go:embed seal
//go:embed git
//go:embed server
//go:embed request
//go:embed proxy
//go:embed grok
//go:embed install
//go:embed deploy-service
//go:embed go
//go:embed cloudflare-proxy
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
