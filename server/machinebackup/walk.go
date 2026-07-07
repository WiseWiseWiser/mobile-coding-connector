package machinebackup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xhd2015/dot-pkgs/go-pkgs/file/detect"
)

type walkResult struct {
	DotFiles      []FileStat
	DirStats      map[string]*DirStat
	Members       []archiveMember
	ExcludedStats excludedStats
}

func discover(home string, rules ExclusionRules) (*walkResult, error) {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil, fmt.Errorf("read home %s: %w", home, err)
	}

	res := &walkResult{
		DirStats:      make(map[string]*DirStat),
		ExcludedStats: newExcludedStats(),
	}

	var dotDirs []string
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasPrefix(name, ".") {
			continue
		}
		if name == "." || name == ".." {
			continue
		}
		rel := normalizeRelPath(name)

		full := filepath.Join(home, name)
		info, err := os.Lstat(full)
		if err != nil {
			return nil, fmt.Errorf("lstat %s: %w", full, err)
		}

		skip, err := shouldSkipPath(home, rel, rules, info.Mode())
		if err != nil {
			return nil, err
		}
		if skip {
			if err := recordSkippedPath(home, rel, rules, info, res.ExcludedStats); err != nil {
				return nil, err
			}
			continue
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			res.DotFiles = append(res.DotFiles, FileStat{Path: rel, Bytes: 0, Symlink: true})
			link, err := os.Readlink(full)
			if err != nil {
				return nil, fmt.Errorf("readlink %s: %w", full, err)
			}
			res.Members = append(res.Members, archiveMember{
				RelPath:   rel,
				Mode:      int64(info.Mode().Perm()),
				Linkname:  link,
				IsSymlink: true,
			})
		case info.IsDir():
			dotDirs = append(dotDirs, name)
		default:
			res.DotFiles = append(res.DotFiles, FileStat{Path: rel, Bytes: info.Size()})
			res.Members = append(res.Members, archiveMember{
				RelPath: rel,
				Mode:    int64(info.Mode().Perm()),
			})
		}
	}

	sort.Slice(res.DotFiles, func(i, j int) bool {
		return res.DotFiles[i].Path < res.DotFiles[j].Path
	})
	sort.Strings(dotDirs)

	for _, dirName := range dotDirs {
		topRel := normalizeRelPath(dirName)
		stat := &DirStat{Path: topRel}
		res.DirStats[topRel] = stat
		if err := walkTree(home, topRel, rules, res, stat); err != nil {
			return nil, err
		}
	}

	sort.Slice(res.Members, func(i, j int) bool {
		return res.Members[i].RelPath < res.Members[j].RelPath
	})
	return res, nil
}

func skipRuleKey(home, rel string, rules ExclusionRules, mode os.FileMode) (string, error) {
	if rules.isIncludedOverride(rel) {
		return "", nil
	}
	if key := rules.ruleKeyForPath(rel); key != "" {
		return key, nil
	}
	if mode.IsRegular() && rules.hasLogSuffix(rel) {
		return logSuffixRule, nil
	}
	if mode.IsRegular() {
		full := filepath.Join(home, filepath.FromSlash(rel))
		isExec, _, err := detect.IsExecutableBinary(full)
		if err != nil {
			return "", fmt.Errorf("detect executable %s: %w", rel, err)
		}
		if isExec {
			return binaryRule, nil
		}
	}
	return "", nil
}

func shouldSkipPath(home, rel string, rules ExclusionRules, mode os.FileMode) (bool, error) {
	key, err := skipRuleKey(home, rel, rules, mode)
	return key != "", err
}

func recordSkippedPath(home, rel string, rules ExclusionRules, info os.FileInfo, stats excludedStats) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	if info.IsDir() {
		return accumulateExcludedTree(home, rel, rules, stats)
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	key, err := skipRuleKey(home, rel, rules, info.Mode())
	if err != nil {
		return err
	}
	if key != "" {
		stats.add(key, 1, info.Size())
	}
	return nil
}

func accumulateExcludedTree(home, rel string, rules ExclusionRules, stats excludedStats) error {
	root := filepath.Join(home, filepath.FromSlash(rel))
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !mode.IsRegular() {
			return nil
		}
		childRel, err := filepath.Rel(home, path)
		if err != nil {
			return fmt.Errorf("rel path %s: %w", path, err)
		}
		key, err := skipRuleKey(home, normalizeRelPath(childRel), rules, mode)
		if err != nil {
			return err
		}
		if key != "" {
			stats.add(key, 1, info.Size())
		}
		return nil
	})
}

func walkTree(home, rel string, rules ExclusionRules, res *walkResult, stat *DirStat) error {
	full := filepath.Join(home, filepath.FromSlash(rel))
	entries, err := os.ReadDir(full)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", full, err)
	}

	for _, ent := range entries {
		childRel := normalizeRelPath(rel + "/" + ent.Name())
		childFull := filepath.Join(home, filepath.FromSlash(childRel))
		info, err := os.Lstat(childFull)
		if err != nil {
			return fmt.Errorf("lstat %s: %w", childFull, err)
		}

		skip, err := shouldSkipPath(home, childRel, rules, info.Mode())
		if err != nil {
			return err
		}
		if skip {
			if err := recordSkippedPath(home, childRel, rules, info, res.ExcludedStats); err != nil {
				return err
			}
			continue
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			stat.Symlinks++
			link, err := os.Readlink(childFull)
			if err != nil {
				return fmt.Errorf("readlink %s: %w", childFull, err)
			}
			res.Members = append(res.Members, archiveMember{
				RelPath:   childRel,
				Mode:      int64(info.Mode().Perm()),
				Linkname:  link,
				IsSymlink: true,
			})
		case info.IsDir():
			stat.Dirs++
			if err := walkTree(home, childRel, rules, res, stat); err != nil {
				return err
			}
		default:
			stat.Files++
			stat.Bytes += info.Size()
			res.Members = append(res.Members, archiveMember{
				RelPath: childRel,
				Mode:    int64(info.Mode().Perm()),
			})
		}
	}
	return nil
}

func sortedDirStats(m map[string]*DirStat) []DirStat {
	paths := make([]string, 0, len(m))
	for p := range m {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	out := make([]DirStat, 0, len(paths))
	for _, p := range paths {
		out = append(out, *m[p])
	}
	return out
}

func dotFilePaths(files []FileStat) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out
}

func totalsFromDotFiles(files []FileStat) SectionTotals {
	var t SectionTotals
	for _, f := range files {
		if f.Symlink {
			t.Symlinks++
		} else {
			t.Files++
		}
		t.Bytes += f.Bytes
	}
	return t
}

func totalsFromDirStats(stats []DirStat) SectionTotals {
	var t SectionTotals
	for _, st := range stats {
		t.Files += st.Files
		t.Symlinks += st.Symlinks
		t.Bytes += st.Bytes
	}
	return t
}

func mergeSectionTotals(parts ...SectionTotals) SectionTotals {
	var t SectionTotals
	for _, p := range parts {
		t.Files += p.Files
		t.Symlinks += p.Symlinks
		t.Bytes += p.Bytes
	}
	return t
}

func allFileStats(home string, res *walkResult) ([]FileStat, error) {
	out := make([]FileStat, 0, len(res.Members))
	for _, m := range res.Members {
		if m.IsSymlink {
			continue
		}
		full := filepath.Join(home, filepath.FromSlash(m.RelPath))
		info, err := os.Stat(full)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", m.RelPath, err)
		}
		if info.IsDir() {
			continue
		}
		out = append(out, FileStat{Path: m.RelPath, Bytes: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func includedPaths(res *walkResult) []string {
	out := make([]string, 0, len(res.DotFiles)+len(res.Members))
	seen := make(map[string]bool)
	add := func(p string) {
		if seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, f := range res.DotFiles {
		add(f.Path)
	}
	for _, m := range res.Members {
		add(m.RelPath)
	}
	sort.Strings(out)
	return out
}