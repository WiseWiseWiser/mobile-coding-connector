package skill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/less-gen/flags"
)

type InstallOptions struct {
	CursorDirName string
	SkillContent  string
}

func HandleInstall(opts InstallOptions, args []string) error {
	var dryRun bool
	var cursor bool
	var codex bool
	var force bool

	args, err := flags.Bool("--dry-run", &dryRun).
		Bool("--cursor", &cursor).
		Bool("--codex", &codex).
		Bool("--force", &force).
		Help("-h,--help", fmt.Sprintf(`
Usage: remote-agent skill install [OPTIONS] [<dir>]

Install the embedded SKILL.md to a directory.

Options:
  --cursor     Install to .cursor/skills/%s
  --codex      Install to .codex/skills/%s
  --force      Overwrite an existing non-empty destination without prompting
  --dry-run    Show what would be created without writing files
`, opts.CursorDirName, opts.CursorDirName)).Parse(args)
	if err != nil {
		return err
	}

	dir, err := resolveInstallDir(opts.CursorDirName, args, cursor, codex)
	if err != nil {
		return err
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	entries, readErr := os.ReadDir(dir)
	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read directory %s: %w", dir, readErr)
	}

	if readErr == nil && len(entries) > 0 {
		if !force && !confirmOverwrite(dir) {
			fmt.Println("Aborted.")
			return nil
		}
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove directory %s: %w", dir, err)
		}
		readErr = os.ErrNotExist
	}

	skillFile := filepath.Join(dir, "SKILL.md")
	if dryRun {
		fmt.Printf("[dry-run] Would create directory: %s\n", dir)
		fmt.Printf("[dry-run] Would create file: %s\n", skillFile)
		return nil
	}

	if readErr != nil {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(skillFile, []byte(opts.SkillContent), 0644); err != nil {
		return fmt.Errorf("write SKILL.md: %w", err)
	}

	fmt.Printf("Installed skill to: %s\n", dir)
	fmt.Printf("  - %s\n", skillFile)
	return nil
}

func resolveInstallDir(cursorDirName string, args []string, cursor bool, codex bool) (string, error) {
	if cursor && codex {
		return "", fmt.Errorf("--cursor and --codex cannot be used together")
	}
	if cursor {
		return filepath.Join(".cursor", "skills", cursorDirName), nil
	}
	if codex {
		return filepath.Join(".codex", "skills", cursorDirName), nil
	}
	if len(args) == 0 {
		return "", fmt.Errorf("install requires a directory path argument or --cursor/--codex flag")
	}
	if len(args) > 1 {
		return "", fmt.Errorf("install accepts at most one directory argument, got %d", len(args))
	}
	return args[0], nil
}

func confirmOverwrite(dir string) bool {
	f, _ := os.Stdin.Stat()
	if f == nil || (f.Mode()&os.ModeCharDevice) == 0 {
		return false
	}
	fmt.Printf("Directory %s is not empty. Overwrite? [y/N] ", dir)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
