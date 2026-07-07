package machinebackup

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ulikunitz/xz"
)

// BuildPlan inspects server home and returns a dry-run backup plan.
func BuildPlan(home string, exclude, include []string, gitOpts GitScanOptions) (*MachineBackupPlan, error) {
	home, err := resolveHome(home)
	if err != nil {
		return nil, err
	}
	rules, err := ResolveExclusionRules(home, exclude, include)
	if err != nil {
		return nil, err
	}
	res, err := discover(home, rules)
	if err != nil {
		return nil, err
	}
	dirStats := sortedDirStats(res.DirStats)
	allFiles, err := allFileStats(home, res)
	if err != nil {
		return nil, err
	}
	dotFilesTotal := totalsFromDotFiles(res.DotFiles)
	dotDirsTotal := totalsFromDirStats(dirStats)
	gitRepos, _, err := ScanGitRepos(home, gitOpts)
	if err != nil {
		return nil, err
	}
	return &MachineBackupPlan{
		Home:          home,
		DotFiles:      res.DotFiles,
		AllFiles:      allFiles,
		DotFilesTotal: dotFilesTotal,
		DirStats:      dirStats,
		DotDirsTotal:  dotDirsTotal,
		GrandTotal:    mergeSectionTotals(dotFilesTotal, dotDirsTotal),
		Excluded:      populateExcludedList(rules, res.ExcludedStats),
		Included:      includedPaths(res),
		GitRepos:      gitRepos,
	}, nil
}

// WriteArchive streams a tar.xz archive of server home dot entries.
func WriteArchive(w io.Writer, home string, exclude, include []string, gitOpts GitScanOptions) error {
	home, err := resolveHome(home)
	if err != nil {
		return err
	}
	rules, err := ResolveExclusionRules(home, exclude, include)
	if err != nil {
		return err
	}
	res, err := discover(home, rules)
	if err != nil {
		return err
	}
	dirStats := sortedDirStats(res.DirStats)
	gitRepos, gitSkipped, err := ScanGitRepos(home, gitOpts)
	if err != nil {
		return err
	}

	manifest := Manifest{
		Version:   manifestVersion,
		CreatedAt: time.Now().UTC(),
		Home:      home,
		Excluded:  rules.ExcludedPaths(),
		DirStats:  dirStats,
		DotFiles:  dotFilePaths(res.DotFiles),
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	xzw, err := xz.NewWriter(w)
	if err != nil {
		return fmt.Errorf("xz writer: %w", err)
	}
	defer xzw.Close()

	tw := tar.NewWriter(xzw)
	defer tw.Close()

	if err := writeTarBytes(tw, "manifest.json", 0644, manifestData); err != nil {
		return err
	}

	for _, member := range res.Members {
		if err := writeMember(tw, home, member); err != nil {
			return err
		}
	}
	return writeBackupMeta(tw, home, rules, gitRepos, gitSkipped)
}

func writeMember(tw *tar.Writer, home string, member archiveMember) error {
	full := filepath.Join(home, filepath.FromSlash(member.RelPath))
	if member.IsSymlink {
		return writeTarSymlink(tw, member.RelPath, member.Linkname, member.Mode)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return fmt.Errorf("read %s: %w", full, err)
	}
	return writeTarBytes(tw, member.RelPath, os.FileMode(member.Mode), data)
}

func writeTarBytes(tw *tar.Writer, name string, mode os.FileMode, data []byte) error {
	hdr := &tar.Header{
		Name:    normalizeRelPath(name),
		Mode:    int64(mode),
		Size:    int64(len(data)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := io.Copy(tw, bytes.NewReader(data))
	return err
}

func writeTarSymlink(tw *tar.Writer, name, target string, mode int64) error {
	hdr := &tar.Header{
		Name:     normalizeRelPath(name),
		Typeflag: tar.TypeSymlink,
		Linkname: target,
		Mode:     mode,
		ModTime:  time.Now().UTC(),
	}
	return tw.WriteHeader(hdr)
}

func resolveHome(home string) (string, error) {
	if home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		return "", fmt.Errorf("HOME is not set")
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat home: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("home is not a directory: %s", abs)
	}
	return abs, nil
}