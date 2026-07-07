package machinebackup

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BuildRestorePlan compares archive entries against home without writing.
func BuildRestorePlan(home string, archive io.Reader, exclude, include []string) (*MachineRestorePlan, error) {
	return planOrApply(home, archive, exclude, include, false)
}

// ApplyRestore writes create/update entries and skips identical paths.
func ApplyRestore(home string, archive io.Reader, exclude, include []string) (*MachineRestorePlan, error) {
	return planOrApply(home, archive, exclude, include, true)
}

func planOrApply(home string, archive io.Reader, exclude, include []string, apply bool) (*MachineRestorePlan, error) {
	home, err := resolveHome(home)
	if err != nil {
		return nil, err
	}
	rules, err := ResolveExclusionRules(home, exclude, include)
	if err != nil {
		return nil, err
	}

	raw, err := io.ReadAll(archive)
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}
	manifest, entries, err := ReadArchive(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	_ = manifest

	plan := &MachineRestorePlan{Home: home}
	for _, ent := range entries {
		rel := normalizeRelPath(ent.Header.Name)
		if rel == "" {
			continue
		}
		target, skip := resolveRestoreTarget(rel)
		if skip {
			continue
		}
		if rules.IsExcluded(target) {
			continue
		}
		action, err := classifyEntry(home, target, ent)
		if err != nil {
			return nil, err
		}
		if action == "skip" {
			plan.Entries = append(plan.Entries, RestoreEntry{Path: target, Action: "skip"})
			continue
		}
		if apply {
			if err := applyEntry(home, target, ent); err != nil {
				return nil, err
			}
		}
		plan.Entries = append(plan.Entries, RestoreEntry{Path: target, Action: action})
	}
	return plan, nil
}

func classifyEntry(home, rel string, ent archiveEntry) (string, error) {
	full := filepath.Join(home, filepath.FromSlash(rel))
	hdr := ent.Header

	if hdr.Typeflag == tar.TypeSymlink {
		existing, err := os.Lstat(full)
		if os.IsNotExist(err) {
			return "create", nil
		}
		if err != nil {
			return "", fmt.Errorf("lstat %s: %w", full, err)
		}
		if existing.Mode()&os.ModeSymlink == 0 {
			return "update", nil
		}
		target, err := os.Readlink(full)
		if err != nil {
			return "", fmt.Errorf("readlink %s: %w", full, err)
		}
		if target == hdr.Linkname {
			return "skip", nil
		}
		return "update", nil
	}

	existing, err := os.Lstat(full)
	if os.IsNotExist(err) {
		return "create", nil
	}
	if err != nil {
		return "", fmt.Errorf("lstat %s: %w", full, err)
	}
	if existing.IsDir() {
		return "skip", nil
	}
	if existing.Mode()&os.ModeSymlink != 0 {
		return "update", nil
	}
	if int64(len(ent.Data)) != existing.Size() {
		return "update", nil
	}
	current, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", full, err)
	}
	if bytes.Equal(current, ent.Data) {
		return "skip", nil
	}
	return "update", nil
}

func applyEntry(home, rel string, ent archiveEntry) error {
	full := filepath.Join(home, filepath.FromSlash(rel))
	hdr := ent.Header

	if hdr.Typeflag == tar.TypeSymlink {
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(full), err)
		}
		_ = os.Remove(full)
		mode := os.FileMode(hdr.Mode)
		if mode == 0 {
			mode = 0777
		}
		if err := os.Symlink(hdr.Linkname, full); err != nil {
			return fmt.Errorf("symlink %s: %w", full, err)
		}
		return os.Chmod(full, mode)
	}

	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(full), err)
	}
	mode := os.FileMode(hdr.Mode)
	if mode == 0 {
		mode = 0644
	}
	if err := os.WriteFile(full, ent.Data, mode); err != nil {
		return fmt.Errorf("write %s: %w", full, err)
	}
	return nil
}