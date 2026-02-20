package lib

import (
	"fmt"
	"os"
	"path/filepath"
)

type SkillSyncOptions struct {
	SourceDir string
	TargetDir string
	DryRun    bool
}

type SkillSyncResult struct {
	SkillsFound  []string
	SkillsSynced []string
}

func SkillSync(opts *SkillSyncOptions) (*SkillSyncResult, error) {
	if _, err := os.Stat(opts.SourceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("source skills directory not found: %s", opts.SourceDir)
	}

	fmt.Printf("Source: %s\n", opts.SourceDir)
	fmt.Printf("Target: %s\n", opts.TargetDir)
	fmt.Println()

	entries, err := os.ReadDir(opts.SourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read source directory: %w", err)
	}

	var skillDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(opts.SourceDir, entry.Name())
		skillMdPath := filepath.Join(skillPath, "SKILL.md")
		if _, err := os.Stat(skillMdPath); err == nil {
			skillDirs = append(skillDirs, entry.Name())
		} else {
			fmt.Printf("Skipping %s (no SKILL.md found)\n", entry.Name())
		}
	}

	result := &SkillSyncResult{
		SkillsFound: skillDirs,
	}

	if len(skillDirs) == 0 {
		fmt.Println("No skills with SKILL.md found in source directory")
		return result, nil
	}

	fmt.Printf("Found %d skill(s) to sync:\n", len(skillDirs))
	for _, name := range skillDirs {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	if opts.DryRun {
		fmt.Println("[DRY RUN] Would clear target directory and copy skills")
		return result, nil
	}

	if _, err := os.Stat(opts.TargetDir); err == nil {
		fmt.Printf("Clearing existing skills in %s...\n", opts.TargetDir)
		entries, err := os.ReadDir(opts.TargetDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read target directory: %w", err)
		}
		for _, entry := range entries {
			path := filepath.Join(opts.TargetDir, entry.Name())
			if err := os.RemoveAll(path); err != nil {
				return nil, fmt.Errorf("failed to remove %s: %w", path, err)
			}
			fmt.Printf("  Removed: %s\n", entry.Name())
		}
	}

	if err := os.MkdirAll(opts.TargetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	fmt.Println("\nCopying skills...")
	result.SkillsSynced = []string{}
	for _, name := range skillDirs {
		srcPath := filepath.Join(opts.SourceDir, name)
		dstPath := filepath.Join(opts.TargetDir, name)

		if err := copySkillDir(srcPath, dstPath); err != nil {
			return nil, fmt.Errorf("failed to copy %s: %w", name, err)
		}
		fmt.Printf("  Copied: %s\n", name)
		result.SkillsSynced = append(result.SkillsSynced, name)
	}

	fmt.Println("\nDone!")
	return result, nil
}

func copySkillDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if entry.Name() == "node_modules" {
				continue
			}
			if err := copySkillDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func GetProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not in a git repository")
}
