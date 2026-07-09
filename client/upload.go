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

// UploadChunkPhase reports what happened for the current chunk event.
type UploadChunkPhase string

const (
	// UploadChunkRetrying means a transient error occurred and the client is
	// about to retry the same chunk.
	UploadChunkRetrying UploadChunkPhase = "retrying"
	// UploadChunkUploaded means the chunk was accepted by the server.
	UploadChunkUploaded UploadChunkPhase = "uploaded"
	// UploadChunkSkipped means the chunk was already present in the server cache.
	UploadChunkSkipped UploadChunkPhase = "skipped"
)

// UploadDirPhase reports directory-level upload events.
type UploadDirPhase string

const (
	// UploadDirPhaseFileStart is emitted before a regular file upload begins.
	UploadDirPhaseFileStart UploadDirPhase = "file_start"
	// UploadDirPhaseDirCreated is emitted when an empty subdirectory is created remotely.
	UploadDirPhaseDirCreated UploadDirPhase = "dir_created"
)

// UploadDirProgress describes progress reported during a directory upload.
type UploadDirProgress struct {
	FileIndex      int
	TotalItems     int
	RelativePath   string
	Phase          UploadDirPhase
	FileSize       int64
	CompletedBytes int64
	TotalBytes     int64
	Chunk          UploadProgress
}

// UploadProgress describes progress reported during a chunked upload.
type UploadProgress struct {
	ChunkIndex     int
	TotalChunks    int
	CompletedBytes int64
	TotalBytes     int64
	// Attempt is the 1-based try for the current chunk (final count when Phase=uploaded).
	Attempt int
	// MaxAttempts is the configured retry cap for a single chunk.
	MaxAttempts int
	Phase       UploadChunkPhase
	// Err is set when Phase=retrying (the error that triggered the retry).
	Err error
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
	ChmodExec  bool
	ChunkRetry *ChunkRetryConfig
	DryRun     bool
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
	stat, err := os.Stat(localFile)
	if err != nil {
		return nil, fmt.Errorf("failed to stat local file: %w", err)
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("local path is a directory, not a file: %s", localFile)
	}

	baseName := filepath.Base(localFile)
	logicalRemote := remotePath
	if logicalRemote == "" {
		logicalRemote = baseName
	} else if strings.HasSuffix(logicalRemote, "/") {
		logicalRemote = logicalRemote + baseName
	}
	resolvedRemote := logicalRemote
	if !strings.HasPrefix(resolvedRemote, "/") {
		home, err := c.GetHome()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve server home dir: %w", err)
		}
		resolvedRemote = strings.TrimRight(home.Home, "/") + "/" + resolvedRemote
	}
	return c.uploadFileResolved(localFile, resolvedRemote, logicalRemote, opts, onProgress)
}

func (c *Client) uploadFileResolved(localFile string, remotePath string, logicalRemote string, opts UploadOptions, onProgress func(UploadProgress)) (*UploadResult, error) {
	stat, err := os.Stat(localFile)
	if err != nil {
		return nil, fmt.Errorf("failed to stat local file: %w", err)
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("local path is a directory, not a file: %s", localFile)
	}

	totalSize := stat.Size()
	if opts.DryRun {
		SimulateUploadChunks(totalSize, ChunkSize, onProgress)
		return &UploadResult{
			Path: logicalRemote,
			Size: totalSize,
		}, nil
	}

	fileHash, chunks, err := computeFileChunkPlan(localFile, ChunkSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read local file: %w", err)
	}

	totalChunks := len(chunks)

	sess, err := c.initUpload(remotePath, fileHash, totalChunks, totalSize, opts)
	if err != nil {
		return nil, fmt.Errorf("init upload failed: %w", err)
	}
	uploadID := sess.UploadID
	received := sess.Received

	completedBytes := int64(0)
	for i, chunkData := range chunks {
		chunkLen := int64(len(chunkData))
		if received[i] {
			completedBytes += chunkLen
			if onProgress != nil {
				onProgress(UploadProgress{
					ChunkIndex:     i,
					TotalChunks:    totalChunks,
					CompletedBytes: completedBytes,
					TotalBytes:     totalSize,
					Phase:          UploadChunkSkipped,
				})
			}
			continue
		}

		attempts, err := c.uploadChunkWithRetry(uploadID, i, chunkData, opts, func(ev UploadProgress) {
			ev.TotalChunks = totalChunks
			ev.TotalBytes = totalSize
			ev.CompletedBytes = completedBytes
			if onProgress != nil {
				onProgress(ev)
			}
		})
		if err != nil && isUploadSessionNotFound(err) {
			sess, initErr := c.initUpload(remotePath, fileHash, totalChunks, totalSize, opts)
			if initErr != nil {
				return nil, fmt.Errorf("re-init upload after session loss failed: %w", initErr)
			}
			uploadID = sess.UploadID
			received = sess.Received
			if received[i] {
				completedBytes += chunkLen
				if onProgress != nil {
					onProgress(UploadProgress{
						ChunkIndex:     i,
						TotalChunks:    totalChunks,
						CompletedBytes: completedBytes,
						TotalBytes:     totalSize,
						Phase:          UploadChunkSkipped,
					})
				}
				continue
			}
			attempts, err = c.uploadChunkWithRetry(uploadID, i, chunkData, opts, func(ev UploadProgress) {
				ev.TotalChunks = totalChunks
				ev.TotalBytes = totalSize
				ev.CompletedBytes = completedBytes
				if onProgress != nil {
					onProgress(ev)
				}
			})
		}
		if err != nil {
			return nil, fmt.Errorf("upload chunk %d failed: %w", i, err)
		}

		completedBytes += chunkLen
		if onProgress != nil {
			onProgress(UploadProgress{
				ChunkIndex:     i,
				TotalChunks:    totalChunks,
				CompletedBytes: completedBytes,
				TotalBytes:     totalSize,
				Attempt:        attempts,
				MaxAttempts:    opts.resolvedChunkRetry().maxAttempts,
				Phase:          UploadChunkUploaded,
			})
		}
	}

	result, err := c.completeUpload(uploadID)
	if err != nil {
		return nil, fmt.Errorf("complete upload failed: %w", err)
	}
	return result, nil
}

type uploadSession struct {
	UploadID string
	Received map[int]bool
}

func (c *Client) initUpload(remotePath, fileHash string, totalChunks int, totalSize int64, opts UploadOptions) (*uploadSession, error) {
	body := map[string]any{
		"path":         remotePath,
		"total_chunks": totalChunks,
		"total_size":   totalSize,
		"chmod_exec":   opts.ChmodExec,
		"file_hash":    fileHash,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(http.MethodPost, "/api/files/upload/init", bytes.NewReader(buf))
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
		UploadID       string `json:"upload_id"`
		ReceivedChunks []int  `json:"received_chunks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode init response: %w", err)
	}
	if out.UploadID == "" {
		return nil, fmt.Errorf("server returned empty upload_id")
	}
	return &uploadSession{
		UploadID: out.UploadID,
		Received: receivedSet(out.ReceivedChunks),
	}, nil
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
		return readUploadAPIError(resp)
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
