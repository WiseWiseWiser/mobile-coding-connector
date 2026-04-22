package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ChunkSize is the per-chunk size used for chunked uploads. It mirrors
// the frontend's CHUNK_SIZE (2 MiB).
const ChunkSize = 2 * 1024 * 1024

// UploadProgress describes progress reported during a chunked upload.
type UploadProgress struct {
	ChunkIndex     int
	TotalChunks    int
	CompletedBytes int64
	TotalBytes     int64
}

// UploadResult is returned on a successful upload, reflecting the server's
// view of the final file.
type UploadResult struct {
	Status string `json:"status"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
}

// UploadOptions configures optional server-side handling for uploads.
type UploadOptions struct {
	ChmodExec bool
}

// UploadFile reads localFile and uploads it to remotePath on the server
// using the server's chunked-upload protocol.
//
// Path resolution rules:
//   - If remotePath is empty, the local file's basename is used.
//   - If remotePath ends with '/', the basename is appended.
//   - If the resolved remotePath is not absolute, it is joined onto the
//     server's home directory (fetched via GetHome).
//
// onProgress may be nil; when set, it is invoked after each chunk completes.
func (c *Client) UploadFile(localFile string, remotePath string, opts UploadOptions, onProgress func(UploadProgress)) (*UploadResult, error) {
	baseName := filepath.Base(localFile)
	if remotePath == "" {
		remotePath = baseName
	} else if strings.HasSuffix(remotePath, "/") {
		remotePath = remotePath + baseName
	}

	// Resolve relative destinations against the server's home dir so the
	// file lands somewhere predictable (~/<path>).
	if !strings.HasPrefix(remotePath, "/") {
		home, err := c.GetHome()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve server home dir: %w", err)
		}
		remotePath = strings.TrimRight(home.Home, "/") + "/" + remotePath
	}

	f, err := os.Open(localFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open local file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat local file: %w", err)
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("local path is a directory, not a file: %s", localFile)
	}

	totalSize := stat.Size()
	totalChunks := int((totalSize + ChunkSize - 1) / ChunkSize)
	if totalChunks < 1 {
		totalChunks = 1
	}

	uploadID, err := c.initUpload(remotePath, totalChunks, totalSize, opts)
	if err != nil {
		return nil, fmt.Errorf("init upload failed: %w", err)
	}

	completedBytes := int64(0)
	buf := make([]byte, ChunkSize)
	for i := 0; i < totalChunks; i++ {
		n, readErr := io.ReadFull(f, buf)
		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			return nil, fmt.Errorf("failed to read chunk %d: %w", i, readErr)
		}

		if err := c.uploadChunk(uploadID, i, buf[:n]); err != nil {
			return nil, fmt.Errorf("upload chunk %d failed: %w", i, err)
		}

		completedBytes += int64(n)
		if onProgress != nil {
			onProgress(UploadProgress{
				ChunkIndex:     i,
				TotalChunks:    totalChunks,
				CompletedBytes: completedBytes,
				TotalBytes:     totalSize,
			})
		}
	}

	result, err := c.completeUpload(uploadID)
	if err != nil {
		return nil, fmt.Errorf("complete upload failed: %w", err)
	}
	return result, nil
}

func (c *Client) initUpload(remotePath string, totalChunks int, totalSize int64, opts UploadOptions) (string, error) {
	body := map[string]any{
		"path":         remotePath,
		"total_chunks": totalChunks,
		"total_size":   totalSize,
		"chmod_exec":   opts.ChmodExec,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := c.NewRequest(http.MethodPost, "/api/files/upload/init", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", readAPIError(resp)
	}

	var out struct {
		UploadID string `json:"upload_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("failed to decode init response: %w", err)
	}
	if out.UploadID == "" {
		return "", fmt.Errorf("server returned empty upload_id")
	}
	return out.UploadID, nil
}

func (c *Client) uploadChunk(uploadID string, chunkIndex int, chunk []byte) error {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	if err := mw.WriteField("upload_id", uploadID); err != nil {
		return err
	}
	if err := mw.WriteField("chunk_index", strconv.Itoa(chunkIndex)); err != nil {
		return err
	}

	part, err := mw.CreateFormFile("chunk", fmt.Sprintf("chunk_%d", chunkIndex))
	if err != nil {
		return err
	}
	if _, err := part.Write(chunk); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}

	req, err := c.NewRequest(http.MethodPost, "/api/files/upload/chunk", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *Client) completeUpload(uploadID string) (*UploadResult, error) {
	body, err := json.Marshal(map[string]string{"upload_id": uploadID})
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(http.MethodPost, "/api/files/upload/complete", bytes.NewReader(body))
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

	var out UploadResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode complete response: %w", err)
	}
	return &out, nil
}
