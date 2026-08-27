package projectpull

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/xgo/support/cmd"
)

// DownloadRequest is the JSON body for POST .../pull-local/download.
type DownloadRequest struct {
	Dir string `json:"dir"`
}

// DownloadManifest is written into the download tar.gz package.
type DownloadManifest struct {
	Commit    string `json:"commit"`
	Branch    string `json:"branch"`
	OriginURL string `json:"origin_url"`
	Mode      string `json:"mode"`
}

// WriteDownloadPackage streams a gzip tar containing:
//   - manifest.json
//   - repo.bundle (git bundle of HEAD history)
//   - files/<rel> for tracked + untracked (non-ignored) worktree files
func WriteDownloadPackage(w io.Writer, dir string) error {
	if err := validateDir(dir); err != nil {
		return err
	}
	insp, err := InspectRepo(dir)
	if err != nil {
		return err
	}

	bundleTmp, err := os.CreateTemp("", "pull-local-dl-bundle-*")
	if err != nil {
		return err
	}
	bundlePath := bundleTmp.Name()
	bundleTmp.Close()
	defer os.Remove(bundlePath)

	if err := cmd.Dir(dir).Run("git", "bundle", "create", bundlePath, "HEAD"); err != nil {
		return fmt.Errorf("git bundle create: %w", err)
	}

	gzw := gzip.NewWriter(w)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	manifest := DownloadManifest{
		Commit:    insp.Commit,
		Branch:    insp.Branch,
		OriginURL: insp.OriginURL,
		Mode:      "download",
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	if err := writeTarBytes(tw, "manifest.json", manifestJSON, 0644); err != nil {
		return err
	}
	if err := writeTarFile(tw, "repo.bundle", bundlePath); err != nil {
		return fmt.Errorf("add repo.bundle: %w", err)
	}

	out, err := cmd.Dir(dir).Output("git", "ls-files", "-z", "-c", "-o", "--exclude-standard")
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}
	for _, rel := range strings.Split(out, "\x00") {
		rel = normalizeRelPath(rel)
		if rel == "" {
			continue
		}
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		info, err := os.Lstat(abs)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := writeTarFile(tw, "files/"+rel, abs); err != nil {
			return fmt.Errorf("add %s: %w", rel, err)
		}
	}
	return nil
}
