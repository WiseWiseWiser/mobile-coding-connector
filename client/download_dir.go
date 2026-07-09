package client

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DownloadDirPhase reports directory-level download events.
type DownloadDirPhase string

const (
	DownloadDirPhaseFileStart DownloadDirPhase = "file_start"
	DownloadDirPhaseDirCreated DownloadDirPhase = "dir_created"
	DownloadDirPhaseDirExists DownloadDirPhase = "dir_exists"
	DownloadDirPhaseDownloading DownloadDirPhase = "downloading"
	DownloadDirPhaseSkipped DownloadDirPhase = "skipped"
	DownloadDirPhaseResumed DownloadDirPhase = "resumed"
	DownloadDirPhaseRetrying DownloadDirPhase = "retrying"
)

// DownloadDirProgress describes progress reported during a directory download.
type DownloadDirProgress struct {
	FileIndex      int
	TotalItems     int
	RelativePath   string
	Phase          DownloadDirPhase
	CompletedBytes int64
	TotalBytes     int64
	FileCompleted  int64
	FileTotal      int64
	Attempt        int
	MaxAttempts    int
	Err            error
}

// DownloadDirResult summarizes a successful directory download.
type DownloadDirResult struct {
	LocalPath    string
	RemotePath   string
	FileCount    int
	TotalSize    int64
	SkippedCount int
	ResumedCount int
}

type remoteDirFile struct {
	remotePath   string
	relativePath string
	size         int64
}

type downloadPlanItemType int

const (
	downloadPlanFile downloadPlanItemType = iota
	downloadPlanEmptyDir
)

type downloadPlanItem struct {
	itemType     downloadPlanItemType
	relativePath string
	file         *remoteDirFile
}

// DownloadDir mirrors remotePath from the server into localDir using per-file GET
// downloads. The local destination may be absent or partially populated (resume).
func (c *Client) DownloadDir(remotePath, localDir string, opts DownloadOptions, onProgress func(DownloadDirProgress)) (*DownloadDirResult, error) {
	absoluteRemote, err := c.ResolveRemoteFilePath(remotePath)
	if err != nil {
		return nil, err
	}

	info, err := c.CheckPath(absoluteRemote)
	if err != nil {
		return nil, fmt.Errorf("failed to check remote path: %w", err)
	}
	if !info.Exists {
		return nil, fmt.Errorf("remote path %q is missing or does not exist", filepath.ToSlash(remotePath))
	}
	if !info.IsDir {
		return nil, fmt.Errorf("remote path %q is not a directory", filepath.ToSlash(remotePath))
	}

	localDir, err = resolveDownloadLocalDir(absoluteRemote, localDir)
	if err != nil {
		return nil, err
	}

	plan, totalSize, err := c.buildDownloadPlan(absoluteRemote)
	if err != nil {
		return nil, err
	}

	totalItems := len(plan)
	completedBytes := int64(0)
	fileCount := 0
	skippedCount := 0
	resumedCount := 0

	for i, item := range plan {
		fileIndex := i + 1
		switch item.itemType {
		case downloadPlanFile:
			fileCount++
			f := item.file
			localPath := filepath.Join(localDir, filepath.FromSlash(f.relativePath))

			if onProgress != nil {
				onProgress(DownloadDirProgress{
					FileIndex:      fileIndex,
					TotalItems:     totalItems,
					RelativePath:   f.relativePath,
					Phase:          DownloadDirPhaseFileStart,
					CompletedBytes: completedBytes,
					TotalBytes:     totalSize,
					FileTotal:      f.size,
				})
			}

			action, err := c.downloadPlanFile(localDir, f, opts, completedBytes, totalSize, fileIndex, totalItems, onProgress, &skippedCount, &resumedCount)
			if err != nil {
				return nil, err
			}
			switch action {
			case downloadActionSkipped:
				completedBytes += f.size
			case downloadActionResumed, downloadActionFull:
				completedBytes += f.size
			}
			_ = localPath

		case downloadPlanEmptyDir:
			localPath := filepath.Join(localDir, filepath.FromSlash(strings.TrimSuffix(item.relativePath, "/")))
			alreadyExists := false
			if st, err := os.Stat(localPath); err == nil && st.IsDir() {
				alreadyExists = true
			} else if err := os.MkdirAll(localPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create local directory %s: %w", localPath, err)
			}
			phase := DownloadDirPhaseDirCreated
			if alreadyExists {
				phase = DownloadDirPhaseDirExists
			}
			if onProgress != nil {
				onProgress(DownloadDirProgress{
					FileIndex:      fileIndex,
					TotalItems:     totalItems,
					RelativePath:   item.relativePath,
					Phase:          phase,
					CompletedBytes: completedBytes,
					TotalBytes:     totalSize,
				})
			}
		}
	}

	return &DownloadDirResult{
		LocalPath:    localDir,
		RemotePath:   absoluteRemote,
		FileCount:    fileCount,
		TotalSize:    totalSize,
		SkippedCount: skippedCount,
		ResumedCount: resumedCount,
	}, nil
}

type downloadAction int

const (
	downloadActionSkipped downloadAction = iota
	downloadActionResumed
	downloadActionFull
)

func (c *Client) downloadPlanFile(
	localDir string,
	f *remoteDirFile,
	opts DownloadOptions,
	completedBytes, totalSize int64,
	fileIndex, totalItems int,
	onProgress func(DownloadDirProgress),
	skippedCount, resumedCount *int,
) (downloadAction, error) {
	localPath := filepath.Join(localDir, filepath.FromSlash(f.relativePath))
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil && filepath.Dir(localPath) != "." {
		return 0, fmt.Errorf("failed to create local directory: %w", err)
	}

	localSize := int64(0)
	localExists := false
	if st, err := osStatRegular(localPath); err == nil {
		localSize = st
		localExists = true
	}

	emit := func(phase DownloadDirPhase, fileCompleted int64, extra func(*DownloadDirProgress)) {
		if onProgress == nil {
			return
		}
		p := DownloadDirProgress{
			FileIndex:      fileIndex,
			TotalItems:     totalItems,
			RelativePath:   f.relativePath,
			Phase:          phase,
			CompletedBytes: completedBytes,
			TotalBytes:     totalSize,
			FileCompleted:  fileCompleted,
			FileTotal:      f.size,
		}
		if extra != nil {
			extra(&p)
		}
		onProgress(p)
	}

	switch {
	case localExists && localSize == f.size:
		*skippedCount++
		emit(DownloadDirPhaseSkipped, f.size, func(p *DownloadDirProgress) {
			p.CompletedBytes = completedBytes + f.size
		})
		return downloadActionSkipped, nil

	case localExists && localSize > 0 && localSize < f.size:
		*resumedCount++
		emit(DownloadDirPhaseResumed, localSize, func(p *DownloadDirProgress) {
			p.CompletedBytes = completedBytes + localSize
		})
		_, err := c.downloadGETWithRetry(f.remotePath, localPath, localSize, false, f.size, opts, func(p DownloadProgress) {
			if onProgress == nil {
				return
			}
			phase := DownloadDirPhaseDownloading
			if p.Phase == DownloadPhaseRetrying {
				phase = DownloadDirPhaseRetrying
			}
			onProgress(DownloadDirProgress{
				FileIndex:      fileIndex,
				TotalItems:     totalItems,
				RelativePath:   f.relativePath,
				Phase:          phase,
				CompletedBytes: completedBytes + p.CompletedBytes,
				TotalBytes:     totalSize,
				FileCompleted:  p.CompletedBytes,
				FileTotal:      f.size,
				Attempt:        p.Attempt,
				MaxAttempts:    p.MaxAttempts,
				Err:            p.Err,
			})
		})
		return downloadActionResumed, err

	case localExists && localSize > f.size:
		emit(DownloadDirPhaseDownloading, 0, nil)
		_, err := c.downloadGETWithRetry(f.remotePath, localPath, 0, true, f.size, opts, func(p DownloadProgress) {
			if onProgress == nil {
				return
			}
			phase := DownloadDirPhaseDownloading
			if p.Phase == DownloadPhaseRetrying {
				phase = DownloadDirPhaseRetrying
			}
			onProgress(DownloadDirProgress{
				FileIndex:      fileIndex,
				TotalItems:     totalItems,
				RelativePath:   f.relativePath,
				Phase:          phase,
				CompletedBytes: completedBytes + p.CompletedBytes,
				TotalBytes:     totalSize,
				FileCompleted:  p.CompletedBytes,
				FileTotal:      f.size,
				Attempt:        p.Attempt,
				MaxAttempts:    p.MaxAttempts,
				Err:            p.Err,
			})
		})
		return downloadActionFull, err

	default:
		_, err := c.downloadGETWithRetry(f.remotePath, localPath, 0, true, f.size, opts, func(p DownloadProgress) {
			if onProgress == nil {
				return
			}
			phase := DownloadDirPhaseDownloading
			if p.Phase == DownloadPhaseRetrying {
				phase = DownloadDirPhaseRetrying
			}
			onProgress(DownloadDirProgress{
				FileIndex:      fileIndex,
				TotalItems:     totalItems,
				RelativePath:   f.relativePath,
				Phase:          phase,
				CompletedBytes: completedBytes + p.CompletedBytes,
				TotalBytes:     totalSize,
				FileCompleted:  p.CompletedBytes,
				FileTotal:      f.size,
				Attempt:        p.Attempt,
				MaxAttempts:    p.MaxAttempts,
				Err:            p.Err,
			})
		})
		return downloadActionFull, err
	}
}

func (c *Client) buildDownloadPlan(absoluteRemote string) (plan []downloadPlanItem, totalSize int64, err error) {
	files, allDirRels, err := c.walkRemoteDownloadTree(absoluteRemote)
	if err != nil {
		return nil, 0, err
	}

	dirsWithFiles := dirsContainingRemoteFiles(files)
	emptyDirs := make(map[string]bool)
	for _, dirRel := range allDirRels {
		if !dirsWithFiles[dirRel] {
			emptyDirs[dirRel] = true
		}
	}

	var visit func(dirPath, relPrefix string) error
	visit = func(dirPath, relPrefix string) error {
		browse, err := c.BrowseDir(dirPath)
		if err != nil {
			return err
		}
		entries := append([]BrowseEntry(nil), browse.Entries...)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name < entries[j].Name
		})
		for _, e := range entries {
			relSlash := filepath.ToSlash(filepath.Join(relPrefix, e.Name))
			if e.IsDir {
				if emptyDirs[relSlash] {
					plan = append(plan, downloadPlanItem{
						itemType:     downloadPlanEmptyDir,
						relativePath: relSlash + "/",
					})
				}
				if err := visit(e.Path, relSlash); err != nil {
					return err
				}
				continue
			}
			totalSize += e.Size
			plan = append(plan, downloadPlanItem{
				itemType:     downloadPlanFile,
				relativePath: relSlash,
				file: &remoteDirFile{
					remotePath:   e.Path,
					relativePath: relSlash,
					size:         e.Size,
				},
			})
		}
		return nil
	}

	if err := visit(absoluteRemote, ""); err != nil {
		return nil, 0, fmt.Errorf("failed to walk remote directory: %w", err)
	}
	return plan, totalSize, nil
}

func (c *Client) walkRemoteDownloadTree(remoteRoot string) (files []remoteDirFile, allDirRels []string, err error) {
	var visit func(dirPath, relPrefix string) error
	visit = func(dirPath, relPrefix string) error {
		browse, err := c.BrowseDir(dirPath)
		if err != nil {
			return err
		}
		entries := append([]BrowseEntry(nil), browse.Entries...)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name < entries[j].Name
		})
		for _, e := range entries {
			relSlash := filepath.ToSlash(filepath.Join(relPrefix, e.Name))
			if e.IsDir {
				allDirRels = append(allDirRels, relSlash)
				if err := visit(e.Path, relSlash); err != nil {
					return err
				}
				continue
			}
			files = append(files, remoteDirFile{
				remotePath:   e.Path,
				relativePath: relSlash,
				size:         e.Size,
			})
		}
		return nil
	}
	if err := visit(remoteRoot, ""); err != nil {
		return nil, nil, fmt.Errorf("failed to browse remote directory: %w", err)
	}
	return files, allDirRels, nil
}

func dirsContainingRemoteFiles(files []remoteDirFile) map[string]bool {
	dirsWithFiles := make(map[string]bool)
	for _, f := range files {
		dir := filepath.ToSlash(filepath.Dir(f.relativePath))
		for dir != "." && dir != "" {
			dirsWithFiles[dir] = true
			dir = filepath.ToSlash(filepath.Dir(dir))
		}
	}
	return dirsWithFiles
}

// CountDownloadDirItems returns item count (files + empty subdirs), file count, and total bytes.
func (c *Client) CountDownloadDirItems(remotePath string) (itemCount int, fileCount int, totalSize int64, err error) {
	absoluteRemote, err := c.ResolveRemoteFilePath(remotePath)
	if err != nil {
		return 0, 0, 0, err
	}
	_, totalSize, err = c.buildDownloadPlan(absoluteRemote)
	if err != nil {
		return 0, 0, 0, err
	}
	files, allDirRels, err := c.walkRemoteDownloadTree(absoluteRemote)
	if err != nil {
		return 0, 0, 0, err
	}
	dirsWithFiles := dirsContainingRemoteFiles(files)
	emptyDirCount := 0
	for _, dirRel := range allDirRels {
		if !dirsWithFiles[dirRel] {
			emptyDirCount++
		}
	}
	return len(files) + emptyDirCount, len(files), totalSize, nil
}

func resolveDownloadLocalDir(absoluteRemote, localPath string) (string, error) {
	baseName := filepath.Base(strings.TrimSuffix(filepath.ToSlash(absoluteRemote), "/"))
	if localPath == "" {
		localPath = baseName
	} else if strings.HasSuffix(localPath, "/") || strings.HasSuffix(localPath, string(os.PathSeparator)) {
		localPath = filepath.Join(strings.TrimSuffix(filepath.ToSlash(localPath), "/"), baseName)
	}
	abs, err := filepath.Abs(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve local directory: %w", err)
	}
	return abs, nil
}