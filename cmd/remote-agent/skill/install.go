package skill

import "github.com/xhd2015/skills/install"

type InstallOptions struct {
	CursorDirName string
	SkillContent  string
}

func HandleInstall(opts InstallOptions, args []string) error {
	return install.HandleInstall(install.InstallOptions{
		CursorDirName: opts.CursorDirName,
		SkillContent:  opts.SkillContent,
		Usage:         "remote-agent skill install",
	}, args)
}
