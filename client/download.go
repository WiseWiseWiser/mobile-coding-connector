package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DownloadPhase reports what happened during a file download event.
type DownloadPhase string

const (
	DownloadPhaseDownloading DownloadPhase = "downloading"
	DownloadPhaseRetrying    DownloadPhase = "retrying"
)

// DownloadProgress describes progress reported during a file download.
type DownloadProgress struct {
	CompletedBytes int64
	TotalBytes     int64
	Phase          DownloadPhase
	Attempt        int
	MaxAttempts    int
	Err            error
}

// DownloadResult is returned on a successful download.
type DownloadResult struct {
	RemotePath string `json:"remote_path"`
	LocalPath  string `json:"local_path"`
	Size       int64  `json:"size"`
}

// ResolveRemoteFilePath normalizes a remote path for file APIs.
//
// Path resolution rules:
//   - ~/path and ~ expand against the server's home directory.
//   - Non-absolute paths are joined onto the server's home directory.
//   - Absolute paths are cleaned and used as-is.
func (c *Client) ResolveRemoteFilePath(remotePath string) (string, error) {
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return "", fmt.Errorf("remote path is required")
	}

	if remotePath == "~" || strings.HasPrefix(remotePath, "~/") || !strings.HasPrefix(remotePath, "/") {
		home, err := c.GetHome()
		if err != nil {
			return "", fmt.Errorf("failed to resolve server home dir: %w", err)
		}
		homeDir := strings.TrimRight(home.Home, "/")
		switch {
		case remotePath == "~":
			return homeDir, nil
		case strings.HasPrefix(remotePath, "~/"):
			return filepath.Join(homeDir, strings.TrimPrefix(remotePath, "~/")), nil
		default:
			return filepath.Join(homeDir, remotePath), nil
		}
	}

	return filepath.Clean(remotePath), nil
}

// DownloadFile downloads remotePath from the server and writes it to localPath.
//
// Path resolution rules:
//   - remotePath follows ResolveRemoteFilePath.
//   - If localPath is empty, the remote file's basename is used in the
//     current working directory.
//
// onProgress may be nil; when set, it is invoked as bytes are written.
func (c *Client) DownloadFile(remotePath, localPath string, opts DownloadOptions, onProgress func(DownloadProgress)) (*DownloadResult, error) {
	resolvedRemote, err := c.ResolveRemoteFilePath(remotePath)
	if err != nil {
		return nil, err
	}

	if localPath == "" {
		localPath = filepath.Base(resolvedRemote)
	}
	if strings.HasSuffix(localPath, string(os.PathSeparator)) || strings.HasSuffix(localPath, "/") {
		localPath = filepath.Join(strings.TrimSuffix(filepath.ToSlash(localPath), "/"), filepath.Base(resolvedRemote))
	}

	var remoteSize int64 = -1
	if info, err := c.CheckPath(resolvedRemote); err == nil && info.Exists && !info.IsDir {
		remoteSize = info.Size
	}

	localSize := int64(0)
	localExists := false
	if st, err := osStatRegular(localPath); err == nil {
		localSize = st
		localExists = true
	}

	startOffset := int64(0)
	truncate := true
	if localExists {
		if remoteSize >= 0 && localSize == remoteSize {
			return &DownloadResult{
				RemotePath: resolvedRemote,
				LocalPath:  localPath,
				Size:       localSize,
			}, nil
		}
		if remoteSize >= 0 && localSize > remoteSize {
			startOffset = 0
			truncate = true
		} else if localSize > 0 {
			startOffset = localSize
			truncate = false
		}
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil && filepath.Dir(localPath) != "." {
		return nil, fmt.Errorf("failed to create local directory: %w", err)
	}

	written, err := c.downloadGETWithRetry(resolvedRemote, localPath, startOffset, truncate, remoteSize, opts, onProgress)
	if err != nil {
		return nil, err
	}

	finalSize := written
	if st, err := osStatRegular(localPath); err == nil {
		finalSize = st
	}

	return &DownloadResult{
		RemotePath: resolvedRemote,
		LocalPath:  localPath,
		Size:       finalSize,
	}, nil
}

func (c *Client) downloadGETOnce(
	resolvedRemote string,
	localPath string,
	startOffset int64,
	truncate bool,
	remoteSize int64,
	onProgress func(DownloadProgress),
) (int64, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/files/download?path="+url.QueryEscape(resolvedRemote), nil)
	if err != nil {
		return 0, err
	}
	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}

	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, readDownloadAPIError(resp)
	}

	totalBytes := remoteSize
	if totalBytes < 0 {
		totalBytes = resp.ContentLength
		if startOffset > 0 && resp.StatusCode == http.StatusPartialContent {
			if cr := resp.Header.Get("Content-Range"); cr != "" {
				if _, end, size, ok := parseContentRange(cr); ok && size >= 0 {
					totalBytes = size
					_ = end
				}
			}
			if totalBytes < 0 {
				totalBytes = startOffset + resp.ContentLength
			}
		}
	}

	flags := os.O_WRONLY | os.O_CREATE
	if truncate && startOffset == 0 {
		flags |= os.O_TRUNC
	}
	dst, err := os.OpenFile(localPath, flags, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open local file: %w", err)
	}
	defer dst.Close()

	if startOffset > 0 {
		if _, err := dst.Seek(startOffset, io.SeekStart); err != nil {
			return 0, fmt.Errorf("failed to seek local file: %w", err)
		}
	}

	writer := io.Writer(dst)
	if onProgress != nil {
		writer = &downloadProgressWriter{
			dst:          dst,
			baseOffset:   startOffset,
			totalBytes:   totalBytes,
			onProgress:   onProgress,
		}
	}

	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to write local file: %w", err)
	}
	finalSize := startOffset + written
	if onProgress != nil {
		if pw, ok := writer.(*downloadProgressWriter); ok {
			if !pw.reported {
				pw.emitProgress(finalSize)
			}
		} else {
			onProgress(DownloadProgress{
				CompletedBytes: finalSize,
				TotalBytes:     totalBytes,
				Phase:          DownloadPhaseDownloading,
			})
		}
	}
	return finalSize, nil
}

func parseContentRange(header string) (start, end, total int64, ok bool) {
	if !strings.HasPrefix(header, "bytes ") {
		return 0, 0, -1, false
	}
	parts := strings.Split(strings.TrimPrefix(header, "bytes "), "/")
	if len(parts) != 2 {
		return 0, 0, -1, false
	}
	total, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		total = -1
	}
	rangeParts := strings.SplitN(parts[0], "-", 2)
	if len(rangeParts) != 2 {
		return 0, 0, total, false
	}
	start, err = strconv.ParseInt(rangeParts[0], 10, 64)
	if err != nil {
		return 0, 0, total, false
	}
	end, err = strconv.ParseInt(rangeParts[1], 10, 64)
	if err != nil {
		return 0, 0, total, false
	}
	return start, end, total, true
}

func osStatRegular(path string) (int64, error) {
	st, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if !st.Mode().IsRegular() {
		return 0, fmt.Errorf("not a regular file")
	}
	return st.Size(), nil
}

type downloadProgressWriter struct {
	dst          io.Writer
	baseOffset   int64
	totalBytes   int64
	completed    int64
	onProgress   func(DownloadProgress)
	lastReported int
	reported     bool
}

func (w *downloadProgressWriter) emitProgress(absolute int64) {
	w.reported = true
	w.onProgress(DownloadProgress{
		CompletedBytes: absolute,
		TotalBytes:     w.totalBytes,
		Phase:          DownloadPhaseDownloading,
	})
}

func (w *downloadProgressWriter) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)
	if n > 0 {
		w.completed += int64(n)
		absolute := w.baseOffset + w.completed
		percent := 0
		if w.totalBytes > 0 {
			percent = int(absolute * 100 / w.totalBytes)
		}
		if percent != w.lastReported || (w.totalBytes > 0 && absolute == w.totalBytes) {
			w.lastReported = percent
			w.emitProgress(absolute)
		}
	}
	return n, err
}