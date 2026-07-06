package machinebackup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type walkResult struct {
	DotFiles []FileStat
	DirStats map[string]*DirStat
	Members  []archiveMember
}

func discover(home string, rules ExclusionRules) (*walkResult, error) {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil, fmt.Errorf("read home %s: %w", home, err)
	}

	res := &walkResult{
		DirStats: make(map[string]*DirStat),
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
		if rules.IsExcluded(rel) || rules.isTopLevelExcluded(name) {
			continue
		}

		full := filepath.Join(home, name)
		info, err := os.Lstat(full)
		if err != nil {
			return nil, fmt.Errorf("lstat %s: %w", full, err)
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

func walkTree(home, rel string, rules ExclusionRules, res *walkResult, stat *DirStat) error {
	full := filepath.Join(home, filepath.FromSlash(rel))
	entries, err := os.ReadDir(full)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", full, err)
	}

	for _, ent := range entries {
		childRel := normalizeRelPath(rel + "/" + ent.Name())
		if rules.IsExcluded(childRel) {
			continue
		}
		childFull := filepath.Join(home, filepath.FromSlash(childRel))
		info, err := os.Lstat(childFull)
		if err != nil {
			return fmt.Errorf("lstat %s: %w", childFull, err)
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