package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// DownloadProgress describes progress reported during a file download.
type DownloadProgress struct {
	CompletedBytes int64
	TotalBytes     int64
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
func (c *Client) DownloadFile(remotePath, localPath string, onProgress func(DownloadProgress)) (*DownloadResult, error) {
	resolvedRemote, err := c.ResolveRemoteFilePath(remotePath)
	if err != nil {
		return nil, err
	}

	if localPath == "" {
		localPath = filepath.Base(resolvedRemote)
	}
	if strings.HasSuffix(localPath, string(os.PathSeparator)) {
		localPath = filepath.Join(localPath, filepath.Base(resolvedRemote))
	}

	req, err := c.NewRequest(http.MethodGet, "/api/files/download?path="+url.QueryEscape(resolvedRemote), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil && filepath.Dir(localPath) != "." {
		return nil, fmt.Errorf("failed to create local directory: %w", err)
	}

	dst, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create local file: %w", err)
	}
	defer dst.Close()

	totalBytes := resp.ContentLength
	writer := io.Writer(dst)
	if onProgress != nil {
		writer = &downloadProgressWriter{
			dst:        dst,
			totalBytes: totalBytes,
			onProgress: onProgress,
		}
	}

	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write local file: %w", err)
	}

	return &DownloadResult{
		RemotePath: resolvedRemote,
		LocalPath:  localPath,
		Size:       written,
	}, nil
}

type downloadProgressWriter struct {
	dst          io.Writer
	totalBytes   int64
	completed    int64
	onProgress   func(DownloadProgress)
	lastReported int
}

func (w *downloadProgressWriter) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)
	if n > 0 {
		w.completed += int64(n)
		percent := 0
		if w.totalBytes > 0 {
			percent = int(w.completed * 100 / w.totalBytes)
		}
		if percent != w.lastReported || w.completed == w.totalBytes {
			w.lastReported = percent
			w.onProgress(DownloadProgress{
				CompletedBytes: w.completed,
				TotalBytes:     w.totalBytes,
			})
		}
	}
	return n, err
}