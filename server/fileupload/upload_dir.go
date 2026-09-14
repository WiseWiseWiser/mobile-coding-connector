package fileupload

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// UploadDirPreflightRequest checks whether applying a directory upload to dest
// would hit override or type conflicts for the given relative file paths.
type UploadDirPreflightRequest struct {
	Dest       string   `json:"dest"`
	Files      []string `json:"files"`
	NoOverride bool     `json:"no_override"`
}

// UploadDirPreflightResponse is returned by /api/files/upload-dir/preflight.
type UploadDirPreflightResponse struct {
	OK             bool     `json:"ok"`
	Conflicts      []string `json:"conflicts,omitempty"`
	TypeConflicts  []string `json:"type_conflicts,omitempty"`
	EffectiveDest  string   `json:"dest"`
}

// UploadDirApplyRequest applies a previously uploaded tar.xz archive into dest.
type UploadDirApplyRequest struct {
	ArchivePath   string `json:"archive_path"`
	Dest          string `json:"dest"`
	NoOverride    bool   `json:"no_override"`
	DeleteArchive bool   `json:"delete_archive"`
}

// UploadDirApplyResponse summarizes a successful apply.
type UploadDirApplyResponse struct {
	Dest       string   `json:"dest"`
	FileCount  int      `json:"file_count"`
	Overridden []string `json:"overridden,omitempty"`
}

func registerUploadDirAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/files/upload-dir/preflight", handleUploadDirPreflight)
	mux.HandleFunc("/api/files/upload-dir/apply", handleUploadDirApply)
}

func handleUploadDirPreflight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req UploadDirPreflightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	dest, err := cleanAbsPath(req.Dest)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	conflicts, typeConflicts, err := preflightUploadDirFiles(dest, req.Files, req.NoOverride)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	ok := len(typeConflicts) == 0 && (!req.NoOverride || len(conflicts) == 0)
	writeJSON(w, UploadDirPreflightResponse{
		OK:            ok,
		Conflicts:     conflicts,
		TypeConflicts: typeConflicts,
		EffectiveDest: dest,
	})
}

func handleUploadDirApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req UploadDirApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := applyUploadDirArchive(req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, resp)
}

func cleanAbsPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("dest is required")
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("path must be absolute: %s", path)
	}
	return clean, nil
}

func preflightUploadDirFiles(dest string, files []string, noOverride bool) (conflicts, typeConflicts []string, err error) {
	for _, rel := range files {
		rel, err = sanitizeTarRelPath(rel)
		if err != nil {
			return nil, nil, err
		}
		full := filepath.Join(dest, filepath.FromSlash(rel))
		info, statErr := os.Lstat(full)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return nil, nil, fmt.Errorf("stat %s: %w", full, statErr)
		}
		if info.IsDir() {
			typeConflicts = append(typeConflicts, rel)
			continue
		}
		if noOverride {
			conflicts = append(conflicts, rel)
		}
	}
	return conflicts, typeConflicts, nil
}

func applyUploadDirArchive(req UploadDirApplyRequest) (*UploadDirApplyResponse, error) {
	dest, err := cleanAbsPath(req.Dest)
	if err != nil {
		return nil, err
	}
	archivePath, err := cleanAbsPath(req.ArchivePath)
	if err != nil {
		return nil, fmt.Errorf("archive_path: %w", err)
	}
	if _, err := os.Stat(archivePath); err != nil {
		return nil, fmt.Errorf("archive not found: %w", err)
	}

	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return nil, fmt.Errorf("mkdir parent %s: %w", parent, err)
	}

	staging, err := os.MkdirTemp(parent, ".upload-dir-staging-*")
	if err != nil {
		return nil, fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(staging)

	fileCount, members, err := extractTarXzToDir(archivePath, staging)
	if err != nil {
		return nil, err
	}

	// Server-side fail-fast guard (TOCTOU): re-check before merge.
	conflicts, typeConflicts, err := preflightUploadDirFiles(dest, members, req.NoOverride)
	if err != nil {
		return nil, err
	}
	if len(typeConflicts) > 0 {
		return nil, fmt.Errorf("type conflict: remote path is a directory where a file is required: %s", strings.Join(typeConflicts, ", "))
	}
	if req.NoOverride && len(conflicts) > 0 {
		return nil, fmt.Errorf("--no-override: %d file(s) would be overwritten: %s", len(conflicts), strings.Join(conflicts, ", "))
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return nil, fmt.Errorf("mkdir dest %s: %w", dest, err)
	}

	overridden, err := mergeStagingIntoDest(staging, dest)
	if err != nil {
		return nil, err
	}

	if req.DeleteArchive {
		_ = os.Remove(archivePath)
	}

	return &UploadDirApplyResponse{
		Dest:       dest,
		FileCount:  fileCount,
		Overridden: overridden,
	}, nil
}

func extractTarXzToDir(archivePath, staging string) (fileCount int, members []string, err error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return 0, nil, fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	xzr, err := xz.NewReader(f)
	if err != nil {
		return 0, nil, fmt.Errorf("xz reader: %w", err)
	}
	tr := tar.NewReader(xzr)

	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return 0, nil, fmt.Errorf("read tar: %w", nextErr)
		}
		rel, err := sanitizeTarRelPath(hdr.Name)
		if err != nil {
			return 0, nil, err
		}
		if rel == "" || rel == "." {
			continue
		}
		target := filepath.Join(staging, filepath.FromSlash(rel))
		if !isWithinDir(staging, target) {
			return 0, nil, fmt.Errorf("tar entry escapes destination: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return 0, nil, fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return 0, nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(target), err)
			}
			mode := os.FileMode(hdr.Mode)
			if mode == 0 {
				mode = 0644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return 0, nil, fmt.Errorf("create %s: %w", target, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return 0, nil, fmt.Errorf("write %s: %w", target, err)
			}
			if err := out.Close(); err != nil {
				return 0, nil, err
			}
			fileCount++
			members = append(members, rel)
		case tar.TypeSymlink:
			return 0, nil, fmt.Errorf("symlinks are not supported in directory upload archives: %s", rel)
		default:
			// Skip other special types.
		}
	}
	return fileCount, members, nil
}

func mergeStagingIntoDest(staging, dest string) (overridden []string, err error) {
	err = filepath.WalkDir(staging, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(staging, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		existing, statErr := os.Lstat(target)
		if statErr == nil {
			if existing.IsDir() {
				return fmt.Errorf("type conflict: remote path is a directory where a file is required: %s", filepath.ToSlash(rel))
			}
			overridden = append(overridden, filepath.ToSlash(rel))
			if err := os.Remove(target); err != nil {
				return fmt.Errorf("remove existing %s: %w", target, err)
			}
		} else if !os.IsNotExist(statErr) {
			return fmt.Errorf("stat %s: %w", target, statErr)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.Rename(path, target); err != nil {
			// Cross-device fallback.
			if copyErr := copyFilePath(path, target); copyErr != nil {
				return fmt.Errorf("move %s to %s: rename: %v; copy: %w", path, target, err, copyErr)
			}
			_ = os.Remove(path)
		}
		return nil
	})
	return overridden, err
}

func copyFilePath(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func sanitizeTarRelPath(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "./")
	name = strings.Trim(name, "/")
	if name == "" {
		return "", nil
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("tar entry must be relative: %s", name)
	}
	parts := strings.Split(name, "/")
	for _, p := range parts {
		if p == ".." {
			return "", fmt.Errorf("tar entry must not contain ..: %s", name)
		}
	}
	return filepath.ToSlash(filepath.Clean(name)), nil
}

func isWithinDir(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
