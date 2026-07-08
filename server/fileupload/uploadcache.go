package fileupload

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const uploadChunkSize = 2 * 1024 * 1024

type uploadMeta struct {
	DestPath    string `json:"dest_path"`
	TotalChunks int    `json:"total_chunks"`
	TotalSize   int64  `json:"total_size"`
	ChmodExec   bool   `json:"chmod_exec"`
}

func isFileHash(id string) bool {
	if len(id) != 64 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func uploadCacheRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, ".ai-critic", "upload-cache")
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	return root, nil
}

func uploadCacheDir(fileHash string) (string, error) {
	if !isFileHash(fileHash) {
		return "", fmt.Errorf("invalid file hash")
	}
	root, err := uploadCacheRoot()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, fileHash)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func chunkFileName(index int, chunkHash string) string {
	return fmt.Sprintf("chunk-%05d-%s", index, chunkHash)
}

func saveUploadMeta(dir string, meta uploadMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
}

func loadUploadMeta(dir string) (uploadMeta, error) {
	data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return uploadMeta{}, err
	}
	var meta uploadMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return uploadMeta{}, err
	}
	return meta, nil
}

func listCachedChunkIndices(dir string) ([]int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var indices []int
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "chunk-") {
			continue
		}
		parts := strings.SplitN(e.Name(), "-", 3)
		if len(parts) < 3 {
			continue
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	return indices, nil
}

func cachedChunkPath(dir string, index int, chunkHash string) string {
	return filepath.Join(dir, chunkFileName(index, chunkHash))
}

func findCachedChunk(dir string, index int) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	prefix := fmt.Sprintf("chunk-%05d-", index)
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		return filepath.Join(dir, e.Name()), true
	}
	return "", false
}

func saveCachedChunk(dir string, index int, data []byte) (string, error) {
	ch := hashBytes(data)
	path := cachedChunkPath(dir, index, ch)
	if _, err := os.Stat(path); err == nil {
		return ch, nil
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return ch, nil
}

func chunkMatchesHash(path string, data []byte) bool {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return hashBytes(existing) == hashBytes(data)
}

func assembleCachedFile(dir string, meta uploadMeta, destPath string) (int64, error) {
	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	var total int64
	for i := 0; i < meta.TotalChunks; i++ {
		path, ok := findCachedChunk(dir, i)
		if !ok {
			return 0, fmt.Errorf("missing chunk %d", i)
		}
		src, err := os.Open(path)
		if err != nil {
			return 0, err
		}
		n, err := io.Copy(dst, src)
		src.Close()
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

func removeUploadCache(dir string) error {
	return os.RemoveAll(dir)
}