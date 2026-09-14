package client

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// localUploadTree describes regular files (and empty dirs) under a local directory.
type localUploadTree struct {
	Files []localUploadFile
	Dirs  []string // relative empty dirs (slash-separated), optional
}

type localUploadFile struct {
	LocalPath    string
	RelativePath string // slash-separated, relative to localDir
	Size         int64
	Mode         os.FileMode
}

func walkLocalUploadContents(localDir string) (*localUploadTree, int64, error) {
	localDir, err := filepath.Abs(localDir)
	if err != nil {
		return nil, 0, err
	}
	tree := &localUploadTree{}
	var totalSize int64
	dirsWithFiles := map[string]bool{}
	allDirs := map[string]bool{}

	err = filepath.WalkDir(localDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if d.IsDir() {
			allDirs[relSlash] = true
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // skip symlinks/devices
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		tree.Files = append(tree.Files, localUploadFile{
			LocalPath:    path,
			RelativePath: relSlash,
			Size:         info.Size(),
			Mode:         info.Mode(),
		})
		totalSize += info.Size()
		dir := filepath.ToSlash(filepath.Dir(rel))
		for dir != "." && dir != "" {
			dirsWithFiles[dir] = true
			parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(dir)))
			if parent == dir {
				break
			}
			dir = parent
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	for dir := range allDirs {
		if !dirsWithFiles[dir] {
			tree.Dirs = append(tree.Dirs, dir)
		}
	}
	return tree, totalSize, nil
}

func packDirTarXz(localDir string, tree *localUploadTree, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer f.Close()

	xzw, err := xz.NewWriter(f)
	if err != nil {
		return fmt.Errorf("xz writer: %w", err)
	}
	tw := tar.NewWriter(xzw)

	for _, dir := range tree.Dirs {
		hdr := &tar.Header{
			Name:     strings.TrimSuffix(dir, "/") + "/",
			Typeflag: tar.TypeDir,
			Mode:     0755,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			_ = tw.Close()
			_ = xzw.Close()
			return err
		}
	}

	for _, file := range tree.Files {
		mode := file.Mode.Perm()
		if mode == 0 {
			mode = 0644
		}
		hdr := &tar.Header{
			Name: file.RelativePath,
			Mode: int64(mode),
			Size: file.Size,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			_ = tw.Close()
			_ = xzw.Close()
			return err
		}
		in, err := os.Open(file.LocalPath)
		if err != nil {
			_ = tw.Close()
			_ = xzw.Close()
			return err
		}
		_, copyErr := io.Copy(tw, in)
		_ = in.Close()
		if copyErr != nil {
			_ = tw.Close()
			_ = xzw.Close()
			return copyErr
		}
	}

	if err := tw.Close(); err != nil {
		_ = xzw.Close()
		return err
	}
	if err := xzw.Close(); err != nil {
		return err
	}
	return f.Close()
}
