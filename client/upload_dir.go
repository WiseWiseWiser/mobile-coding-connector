package client

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// UploadDirResult summarizes a successful directory upload.
type UploadDirResult struct {
	Path      string
	FileCount int
	TotalSize int64
}

type localDirFile struct {
	localPath  string
	remotePath string
	chmodExec  bool
	size       int64
}

type uploadPlanItemType int

const (
	uploadPlanFile uploadPlanItemType = iota
	uploadPlanEmptyDir
)

type uploadPlanItem struct {
	itemType     uploadPlanItemType
	relativePath string
	file         *localDirFile
}

// UploadDir mirrors localDir onto remotePath on the server using per-file chunked
// uploads. The remote destination must be missing or a completely empty directory.
func (c *Client) UploadDir(localDir, remotePath string, opts UploadOptions, onProgress func(UploadDirProgress)) (*UploadDirResult, error) {
	localDir, err := filepath.Abs(localDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve local directory: %w", err)
	}
	info, err := os.Stat(localDir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat local directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local path is not a directory: %s", localDir)
	}

	logicalRemote, absoluteRemote, err := c.resolveRemoteDir(localDir, remotePath)
	if err != nil {
		return nil, err
	}
	if err := c.guardUploadDirDestination(absoluteRemote); err != nil {
		return nil, err
	}

	plan, subdirs, totalSize, err := buildUploadPlan(localDir, absoluteRemote)
	if err != nil {
		return nil, err
	}

	if len(subdirs) > 0 {
		if err := c.mkdirRemote(subdirs...); err != nil {
			return nil, fmt.Errorf("failed to create remote directories: %w", err)
		}
	}

	totalItems := len(plan)
	completedBytes := int64(0)
	fileCount := 0

	for i, item := range plan {
		fileIndex := i + 1
		switch item.itemType {
		case uploadPlanFile:
			fileCount++
			f := item.file
			if onProgress != nil {
				onProgress(UploadDirProgress{
					FileIndex:      fileIndex,
					TotalItems:     totalItems,
					RelativePath:   item.relativePath,
					Phase:          UploadDirPhaseFileStart,
					FileSize:       f.size,
					CompletedBytes: completedBytes,
					TotalBytes:     totalSize,
				})
			}

			priorCompleted := completedBytes
			if _, err := c.uploadFileResolved(f.localPath, f.remotePath, UploadOptions{
				ChmodExec:  f.chmodExec,
				ChunkRetry: opts.ChunkRetry,
			}, func(chunk UploadProgress) {
				if onProgress == nil {
					return
				}
				onProgress(UploadDirProgress{
					FileIndex:      fileIndex,
					TotalItems:     totalItems,
					RelativePath:   item.relativePath,
					CompletedBytes: priorCompleted + chunk.CompletedBytes,
					TotalBytes:     totalSize,
					Chunk:          chunk,
				})
			}); err != nil {
				return nil, err
			}
			completedBytes += f.size

		case uploadPlanEmptyDir:
			if onProgress != nil {
				onProgress(UploadDirProgress{
					FileIndex:      fileIndex,
					TotalItems:     totalItems,
					RelativePath:   item.relativePath,
					Phase:          UploadDirPhaseDirCreated,
					CompletedBytes: completedBytes,
					TotalBytes:     totalSize,
				})
			}
		}
	}

	return &UploadDirResult{
		Path:      logicalRemote,
		FileCount: fileCount,
		TotalSize: totalSize,
	}, nil
}

func (c *Client) guardUploadDirDestination(remoteDir string) error {
	info, err := c.CheckPath(remoteDir)
	if err != nil {
		return fmt.Errorf("failed to check upload destination: %w", err)
	}
	if !info.Exists {
		return nil
	}
	if !info.IsDir {
		return fmt.Errorf(
			"upload destination %q already exists and is not a directory; it must be missing or a completely empty directory",
			filepath.ToSlash(remoteDir),
		)
	}
	browse, err := c.BrowseDir(remoteDir)
	if err != nil {
		return fmt.Errorf("failed to inspect upload destination: %w", err)
	}
	if len(browse.Entries) > 0 {
		return fmt.Errorf(
			"upload destination %q is not empty; it must be missing or a completely empty directory",
			filepath.ToSlash(remoteDir),
		)
	}
	return nil
}

func buildUploadPlan(localDir, absoluteRemote string) (plan []uploadPlanItem, subdirs []string, totalSize int64, err error) {
	files, allDirRels, err := walkLocalUploadTree(localDir, absoluteRemote)
	if err != nil {
		return nil, nil, 0, err
	}

	dirsWithFiles := dirsContainingFiles(localDir, files)
	emptyDirs := make(map[string]bool)
	for _, dirRel := range allDirRels {
		if !dirsWithFiles[dirRel] {
			emptyDirs[dirRel] = true
		}
	}

	err = filepath.WalkDir(localDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == localDir {
			return nil
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)

		if d.IsDir() {
			if emptyDirs[relSlash] {
				plan = append(plan, uploadPlanItem{
					itemType:     uploadPlanEmptyDir,
					relativePath: relSlash + "/",
				})
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		stat, err := d.Info()
		if err != nil {
			return err
		}
		if !stat.Mode().IsRegular() {
			return nil
		}

		var matched *localDirFile
		for i := range files {
			if files[i].localPath == path {
				matched = &files[i]
				break
			}
		}
		if matched == nil {
			return fmt.Errorf("internal error: missing upload plan file for %s", relSlash)
		}
		totalSize += matched.size
		plan = append(plan, uploadPlanItem{
			itemType:     uploadPlanFile,
			relativePath: relSlash,
			file:         matched,
		})
		return nil
	})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to walk local directory: %w", err)
	}
	return plan, subdirsFromFilesAndDirs(localDir, absoluteRemote, files, allDirRels), totalSize, nil
}

func walkLocalUploadTree(localDir, absoluteRemote string) (files []localDirFile, allDirRels []string, err error) {
	err = filepath.WalkDir(localDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == localDir {
			return nil
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		if d.IsDir() {
			allDirRels = append(allDirRels, relSlash)
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		stat, err := d.Info()
		if err != nil {
			return err
		}
		if !stat.Mode().IsRegular() {
			return nil
		}
		files = append(files, localDirFile{
			localPath:  path,
			remotePath: filepath.Join(absoluteRemote, relSlash),
			chmodExec:  stat.Mode()&0o111 != 0,
			size:       stat.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to walk local directory: %w", err)
	}
	return files, allDirRels, nil
}

func dirsContainingFiles(localDir string, files []localDirFile) map[string]bool {
	dirsWithFiles := make(map[string]bool)
	for _, f := range files {
		rel, err := filepath.Rel(localDir, f.localPath)
		if err != nil {
			continue
		}
		relSlash := filepath.ToSlash(rel)
		dir := filepath.ToSlash(filepath.Dir(relSlash))
		for dir != "." && dir != "" {
			dirsWithFiles[dir] = true
			dir = filepath.ToSlash(filepath.Dir(dir))
		}
	}
	return dirsWithFiles
}

func subdirsFromFilesAndDirs(localDir, absoluteRemote string, files []localDirFile, allDirRels []string) []string {
	seen := make(map[string]bool)
	var subdirs []string
	add := func(rel string) {
		if rel == "" || seen[rel] {
			return
		}
		seen[rel] = true
		subdirs = append(subdirs, filepath.Join(absoluteRemote, rel))
	}
	for _, dirRel := range allDirRels {
		add(dirRel)
	}
	for _, f := range files {
		rel, err := filepath.Rel(localDir, f.localPath)
		if err != nil {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(rel))
		for dir != "." && dir != "" {
			add(dir)
			dir = filepath.ToSlash(filepath.Dir(dir))
		}
	}
	return subdirs
}

// CountUploadDirItems returns item count (files + empty subdirs), file count, and total bytes.
func CountUploadDirItems(localDir string) (itemCount int, fileCount int, totalSize int64, err error) {
	files, allDirRels, err := walkLocalUploadTree(localDir, "")
	if err != nil {
		return 0, 0, 0, err
	}
	dirsWithFiles := dirsContainingFiles(localDir, files)
	emptyDirCount := 0
	for _, dirRel := range allDirRels {
		if !dirsWithFiles[dirRel] {
			emptyDirCount++
		}
	}
	for _, f := range files {
		totalSize += f.size
	}
	return len(files) + emptyDirCount, len(files), totalSize, nil
}

func (c *Client) mkdirRemote(paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	argv := append([]string{"mkdir", "-p"}, paths...)
	code, err := c.Exec(ExecRequest{Argv: argv}, nil)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("mkdir exited with status %d", code)
	}
	return nil
}