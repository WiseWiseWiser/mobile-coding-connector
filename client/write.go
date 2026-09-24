package client

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

// WriteResult is returned by WriteFileConditional on success.
type WriteResult struct {
	// Path is the requested remote path.
	Path string `json:"path"`
	// ResolvedPath differs from Path only when the write went through a
	// symlink; the link itself is preserved and its target was replaced.
	ResolvedPath string `json:"resolved_path"`
	Size         int64  `json:"size"`
	MD5          string `json:"md5"`
	// Created reports that the file did not exist before this write.
	Created bool   `json:"created"`
	Mode    string `json:"mode,omitempty"`
}

// FileConflictError reports a rejected conditional write: the remote file no
// longer has the md5 the caller expected, so nothing was written.
type FileConflictError struct {
	Path           string
	ExpectedMD5    string
	CurrentMD5     string
	CurrentExists  bool
	CurrentSize    int64
	CurrentModTime string
}

func (e *FileConflictError) Error() string {
	return fmt.Sprintf("file changed since it was read: %s (expected md5 %s, current md5 %s)",
		e.Path, e.ExpectedMD5, e.CurrentMD5)
}

// UnsupportedWriteError reports that the server did not route the conditional
// write endpoint, which means it predates `remote-agent edit` support.
type UnsupportedWriteError struct {
	Status int
	Body   string
}

func (e *UnsupportedWriteError) Error() string {
	msg := fmt.Sprintf("server does not support conditional writes (HTTP %d)", e.Status)
	if snippet := strings.TrimSpace(e.Body); snippet != "" {
		msg += ": " + snippet
	}
	return msg
}

// WriteFileConditional uploads localPath to remotePath, but only when the
// remote file's md5 still equals expectedMD5. An empty expectedMD5 means "no
// precondition".
//
// A missing remote file counts as empty content (md5 of zero bytes), so a
// caller that downloaded a non-existent path can create it by passing the
// empty-content md5. On mismatch the server writes nothing and the returned
// error is a *FileConflictError.
func (c *Client) WriteFileConditional(remotePath, localPath, expectedMD5 string) (*WriteResult, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local file: %w", err)
	}
	defer f.Close()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		writeErr := writeConditionalForm(mw, f, remotePath, expectedMD5)
		if closeErr := mw.Close(); writeErr == nil {
			writeErr = closeErr
		}
		_ = pw.CloseWithError(writeErr)
	}()

	req, err := c.NewRequest(http.MethodPost, "/api/files/write", pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read write response: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusConflict:
		var conflict struct {
			Path           string `json:"path"`
			ExpectedMD5    string `json:"expected_md5"`
			CurrentMD5     string `json:"current_md5"`
			CurrentExists  bool   `json:"current_exists"`
			CurrentSize    int64  `json:"current_size"`
			CurrentModTime string `json:"current_mod_time"`
		}
		if err := json.Unmarshal(body, &conflict); err != nil {
			return nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		return nil, &FileConflictError{
			Path:           conflict.Path,
			ExpectedMD5:    conflict.ExpectedMD5,
			CurrentMD5:     conflict.CurrentMD5,
			CurrentExists:  conflict.CurrentExists,
			CurrentSize:    conflict.CurrentSize,
			CurrentModTime: conflict.CurrentModTime,
		}
	case resp.StatusCode == http.StatusNotFound:
		return nil, &UnsupportedWriteError{Status: resp.StatusCode, Body: string(body)}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, fmt.Errorf("%s: %s", resp.Status, apiErrorMessage(body))
	}

	var out WriteResult
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to decode write response: %w", err)
	}
	return &out, nil
}

func writeConditionalForm(mw *multipart.Writer, content io.Reader, remotePath, expectedMD5 string) error {
	if err := mw.WriteField("path", remotePath); err != nil {
		return err
	}
	if expectedMD5 != "" {
		if err := mw.WriteField("expected_md5", expectedMD5); err != nil {
			return err
		}
	}
	part, err := mw.CreateFormFile("file", "content")
	if err != nil {
		return err
	}
	_, err = io.Copy(part, content)
	return err
}

// apiErrorMessage extracts the {"error": ...} field of an API error body,
// falling back to the raw body.
func apiErrorMessage(body []byte) string {
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return errResp.Error
	}
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	return snippet
}
