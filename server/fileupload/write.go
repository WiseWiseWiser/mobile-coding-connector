package fileupload

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WritePath is the conditional single-file write endpoint. Clients pass the
// content they want to store plus the md5 they believe the file currently has;
// the write is rejected with 409 Conflict when the file changed since then.
//
// This is the write half of `remote-agent edit`: the CLI downloads a file,
// opens it in a local editor, and writes it back only when nobody else touched
// it meanwhile.
const WritePath = "/api/files/write"

// EmptyMD5 is the md5 of zero bytes. A missing file is treated as empty
// content, so this is the value clients send to require "the file still does
// not exist" and the value the server reports for an absent file.
const EmptyMD5 = "d41d8cd98f00b204e9800998ecf8427e"

// maxWriteBody caps the multipart body accepted by handleWrite (mirrors the
// limit used by the chunk-less upload endpoint).
const maxWriteBody = 100 << 20

// WriteResponse is the 200 body of WritePath.
type WriteResponse struct {
	Status string `json:"status"`
	// Path is the requested (cleaned) path.
	Path string `json:"path"`
	// ResolvedPath is Path with a trailing symlink chain resolved. It differs
	// from Path only when the write went through a symlink; the symlink itself
	// is preserved.
	ResolvedPath string `json:"resolved_path"`
	Size         int64  `json:"size"`
	MD5          string `json:"md5"`
	// Created is true when the file did not exist before this write.
	Created bool   `json:"created"`
	Mode    string `json:"mode,omitempty"`
}

// WriteConflictResponse is the 409 body of WritePath.
type WriteConflictResponse struct {
	Error         string `json:"error"`
	Path          string `json:"path"`
	ResolvedPath  string `json:"resolved_path"`
	ExpectedMD5   string `json:"expected_md5"`
	CurrentMD5    string `json:"current_md5"`
	CurrentExists bool   `json:"current_exists"`
	CurrentSize   int64  `json:"current_size"`
	CurrentMod    string `json:"current_mod_time,omitempty"`
}

// writeJSONStatus writes data as JSON with an explicit status code.
func writeJSONStatus(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// handleWrite implements POST WritePath.
//
// Form fields (multipart/form-data):
//
//	path          destination path (required)
//	expected_md5  md5 the file must currently have; empty means "no precondition"
//	file          new content (required)
//
// A missing file is treated as empty content (EmptyMD5), so a client that
// downloaded a non-existent path can still create it. A mismatch returns 409
// with both hashes and never touches the file.
func handleWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(maxWriteBody); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse form: %v", err))
		return
	}

	destPath := r.FormValue("path")
	if destPath == "" {
		writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}
	destPath = filepath.Clean(destPath)
	expectedMD5 := strings.ToLower(strings.TrimSpace(r.FormValue("expected_md5")))

	resolved, err := resolveWritePath(destPath, 0)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("failed to resolve path: %v", err))
		return
	}

	info, statErr := os.Stat(resolved)
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stat file: %v", statErr))
		return
	}
	exists := statErr == nil
	if exists && info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, "path is a directory")
		return
	}

	currentMD5 := EmptyMD5
	var currentSize int64
	var currentMod time.Time
	if exists {
		currentSize = info.Size()
		currentMod = info.ModTime()
		currentMD5, err = fileMD5(resolved)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to hash current file: %v", err))
			return
		}
	}

	if expectedMD5 != "" && currentMD5 != expectedMD5 {
		writeJSONStatus(w, http.StatusConflict, WriteConflictResponse{
			Error:         "file changed since it was read",
			Path:          destPath,
			ResolvedPath:  resolved,
			ExpectedMD5:   expectedMD5,
			CurrentMD5:    currentMD5,
			CurrentExists: exists,
			CurrentSize:   currentSize,
			CurrentMod:    formatModTime(currentMod),
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("file is required: %v", err))
		return
	}
	defer file.Close()

	dir := filepath.Dir(resolved)
	if err := os.MkdirAll(dir, 0755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create directory: %v", err))
		return
	}

	// Write to a sibling temp file, then rename over the target: readers never
	// observe a half-written file, and the destination keeps working when the
	// content is shorter than before.
	tmp, err := os.CreateTemp(dir, ".remote-agent-write-*")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create temp file: %v", err))
		return
	}
	tmpName := tmp.Name()
	discard := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	written, err := io.Copy(tmp, file)
	if err != nil {
		discard()
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write file: %v", err))
		return
	}
	if err := tmp.Sync(); err != nil {
		discard()
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to flush file: %v", err))
		return
	}

	mode := os.FileMode(0644)
	if exists {
		mode = info.Mode().Perm()
	}
	if err := tmp.Chmod(mode); err != nil {
		discard()
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to set mode: %v", err))
		return
	}
	if exists {
		preserveOwner(tmpName, info)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to close temp file: %v", err))
		return
	}
	if err := os.Rename(tmpName, resolved); err != nil {
		_ = os.Remove(tmpName)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to replace file: %v", err))
		return
	}

	newMD5, err := fileMD5(resolved)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to hash written file: %v", err))
		return
	}

	writeJSON(w, WriteResponse{
		Status:       "ok",
		Path:         destPath,
		ResolvedPath: resolved,
		Size:         written,
		MD5:          newMD5,
		Created:      !exists,
		Mode:         mode.String(),
	})
}

func formatModTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// resolveWritePath returns the path a write should land on, following a
// symlink chain in the final component so that writing through a symlink
// replaces the link target instead of the link. Missing trailing components
// are kept as-is (their parent chain is still resolved), which is how a new
// file under a symlinked directory is handled.
//
// depth bounds symlink cycles; it also covers EvalSymlinks-style loops that
// Lstat cannot detect.
func resolveWritePath(path string, depth int) (string, error) {
	if depth > 40 {
		return "", fmt.Errorf("too many levels of symbolic links: %s", path)
	}

	info, err := os.Lstat(path)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink == 0 {
			return path, nil
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		return resolveWritePath(target, depth+1)
	case !errors.Is(err, fs.ErrNotExist):
		return "", err
	}

	parent := filepath.Dir(path)
	if parent == path {
		// Reached the filesystem root without finding an existing component.
		return path, nil
	}
	resolvedParent, err := resolveWritePath(parent, depth+1)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolvedParent, filepath.Base(path)), nil
}

// fileMD5 returns the hex md5 of the file at path.
func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
