package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

const help = `Usage: go run ./script/skills/sync/cursor [options]

Syncs skills from the project's skills/ directory to .cursor/skills/
for use with Cursor's per-project skills feature.

This script:
1. Clears all existing skills in .cursor/skills/
2. Copies all skill directories from skills/ to .cursor/skills/

Options:
  -h, --help     Show this help message
  --dry-run      Show what would be done without making changes
`

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	dryRun := false
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Print(help)
			return nil
		case "--dry-run":
			dryRun = true
		default:
			return fmt.Errorf("unknown argument: %s", arg)
		}
	}

	projectRoot, err := lib.GetProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	_, err = lib.SkillSync(&lib.SkillSyncOptions{
		SourceDir: filepath.Join(projectRoot, "skills"),
		TargetDir: filepath.Join(projectRoot, ".cursor", "skills"),
		DryRun:    dryRun,
	})

	return err
}
