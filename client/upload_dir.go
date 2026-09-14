package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// UploadDirResult summarizes a successful directory upload.
type UploadDirResult struct {
	Path        string
	FileCount   int
	TotalSize   int64 // uncompressed local bytes
	ArchiveSize int64
	Overridden  []string
}

// UploadDir mirrors localDir onto the remote server using a local tar.xz pack,
// one chunked archive upload, and a remote extract/merge apply step.
//
// Destination resolution follows cp -R rules (see ResolveEffectiveUploadDir).
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

	home, err := c.GetHome()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server home dir: %w", err)
	}
	remoteTarget := ResolveUploadDirTarget(localDir, remotePath, home.Home)

	targetInfo, err := c.CheckPath(remoteTarget)
	if err != nil {
		return nil, fmt.Errorf("failed to check upload destination: %w", err)
	}
	effective, err := ResolveEffectiveUploadDir(localDir, remoteTarget, targetInfo)
	if err != nil {
		return nil, err
	}
	if targetInfo != nil && targetInfo.Exists && targetInfo.IsDir {
		effInfo, err := c.CheckPath(effective)
		if err != nil {
			return nil, fmt.Errorf("failed to check nested upload destination: %w", err)
		}
		if effInfo.Exists && !effInfo.IsDir {
			return nil, fmt.Errorf("upload destination %q is a file; refusing directory upload", filepath.ToSlash(effective))
		}
	}

	tree, totalSize, err := walkLocalUploadContents(localDir)
	if err != nil {
		return nil, err
	}
	relFiles := make([]string, 0, len(tree.Files))
	for _, f := range tree.Files {
		relFiles = append(relFiles, f.RelativePath)
	}

	preflight, err := c.UploadDirPreflight(effective, relFiles, opts.NoOverride)
	if err != nil {
		return nil, err
	}
	if len(preflight.TypeConflicts) > 0 {
		return nil, fmt.Errorf("type conflict: remote path is a directory where a file is required: %s", strings.Join(preflight.TypeConflicts, ", "))
	}
	if opts.NoOverride && len(preflight.Conflicts) > 0 {
		return nil, fmt.Errorf("--no-override: %d file(s) would be overwritten:\n  %s",
			len(preflight.Conflicts), strings.Join(preflight.Conflicts, "\n  "))
	}

	if onProgress != nil {
		onProgress(UploadDirProgress{
			Phase:          UploadDirPhaseResolved,
			RelativePath:   filepath.ToSlash(effective),
			TotalItems:     len(tree.Files) + len(tree.Dirs),
			TotalBytes:     totalSize,
			CompletedBytes: 0,
		})
	}

	if opts.DryRun {
		// Emit remaining stages so CLI can print a stable [n/4] spine with would:.
		if onProgress != nil {
			onProgress(UploadDirProgress{Phase: UploadDirPhasePacking, TotalBytes: totalSize})
			onProgress(UploadDirProgress{Phase: UploadDirPhaseUploading, TotalBytes: totalSize})
			onProgress(UploadDirProgress{
				Phase:        UploadDirPhaseApplying,
				RelativePath: filepath.ToSlash(effective),
			})
		}
		return &UploadDirResult{
			Path:      filepath.ToSlash(effective),
			FileCount: len(tree.Files),
			TotalSize: totalSize,
		}, nil
	}

	tmpArchive, err := os.CreateTemp("", "remote-agent-upload-dir-*.tar.xz")
	if err != nil {
		return nil, fmt.Errorf("create local archive: %w", err)
	}
	archiveLocal := tmpArchive.Name()
	_ = tmpArchive.Close()
	defer os.Remove(archiveLocal)

	if onProgress != nil {
		onProgress(UploadDirProgress{Phase: UploadDirPhasePacking, TotalBytes: totalSize})
	}
	if err := packDirTarXz(localDir, tree, archiveLocal); err != nil {
		return nil, fmt.Errorf("pack directory archive: %w", err)
	}
	archiveInfo, err := os.Stat(archiveLocal)
	if err != nil {
		return nil, err
	}
	if onProgress != nil {
		onProgress(UploadDirProgress{
			Phase:    UploadDirPhasePacking,
			FileSize: archiveInfo.Size(),
		})
	}

	remoteArchive := filepath.ToSlash(filepath.Join(home.Home, ".ai-critic", "upload-dir-tmp", filepath.Base(archiveLocal)))

	if onProgress != nil {
		onProgress(UploadDirProgress{
			Phase:      UploadDirPhaseUploading,
			FileSize:   archiveInfo.Size(),
			TotalBytes: archiveInfo.Size(),
		})
	}

	uploadOpts := UploadOptions{
		NoCompress: true, // already xz-compressed
		ChunkRetry: opts.ChunkRetry,
	}
	if _, err := c.UploadFile(archiveLocal, remoteArchive, uploadOpts, func(chunk UploadProgress) {
		if onProgress == nil {
			return
		}
		onProgress(UploadDirProgress{
			Phase:          UploadDirPhaseUploading,
			CompletedBytes: chunk.CompletedBytes,
			TotalBytes:     chunk.TotalBytes,
			FileSize:       archiveInfo.Size(),
			Chunk:          chunk,
		})
	}); err != nil {
		return nil, fmt.Errorf("upload archive: %w", err)
	}

	if onProgress != nil {
		onProgress(UploadDirProgress{Phase: UploadDirPhaseApplying, RelativePath: filepath.ToSlash(effective)})
	}
	apply, err := c.UploadDirApply(remoteArchive, effective, opts.NoOverride, true)
	if err != nil {
		return nil, err
	}

	return &UploadDirResult{
		Path:        apply.Dest,
		FileCount:   apply.FileCount,
		TotalSize:   totalSize,
		ArchiveSize: archiveInfo.Size(),
		Overridden:  apply.Overridden,
	}, nil
}

// UploadDirPreflightResult is returned by UploadDirPreflight.
type UploadDirPreflightResult struct {
	OK            bool
	Conflicts     []string
	TypeConflicts []string
	Dest          string
}

func (c *Client) UploadDirPreflight(dest string, files []string, noOverride bool) (*UploadDirPreflightResult, error) {
	body, err := json.Marshal(map[string]any{
		"dest":        dest,
		"files":       files,
		"no_override": noOverride,
	})
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/files/upload-dir/preflight", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}
	var out struct {
		OK            bool     `json:"ok"`
		Conflicts     []string `json:"conflicts"`
		TypeConflicts []string `json:"type_conflicts"`
		Dest          string   `json:"dest"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &UploadDirPreflightResult{
		OK:            out.OK,
		Conflicts:     out.Conflicts,
		TypeConflicts: out.TypeConflicts,
		Dest:          out.Dest,
	}, nil
}

// UploadDirApplyResult is returned by UploadDirApply.
type UploadDirApplyResult struct {
	Dest       string
	FileCount  int
	Overridden []string
}

func (c *Client) UploadDirApply(archivePath, dest string, noOverride, deleteArchive bool) (*UploadDirApplyResult, error) {
	body, err := json.Marshal(map[string]any{
		"archive_path":   archivePath,
		"dest":           dest,
		"no_override":    noOverride,
		"delete_archive": deleteArchive,
	})
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/files/upload-dir/apply", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}
	var out struct {
		Dest       string   `json:"dest"`
		FileCount  int      `json:"file_count"`
		Overridden []string `json:"overridden"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &UploadDirApplyResult{
		Dest:       out.Dest,
		FileCount:  out.FileCount,
		Overridden: out.Overridden,
	}, nil
}

// CountUploadDirItems returns item count (files + empty subdirs), file count, and total bytes.
func CountUploadDirItems(localDir string) (itemCount int, fileCount int, totalSize int64, err error) {
	tree, totalSize, err := walkLocalUploadContents(localDir)
	if err != nil {
		return 0, 0, 0, err
	}
	return len(tree.Files) + len(tree.Dirs), len(tree.Files), totalSize, nil
}
