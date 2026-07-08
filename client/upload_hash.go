package client

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
)

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func computeFileChunkPlan(path string, chunkSize int) (fileHash string, chunks [][]byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	hasher := sha256.New()
	buf := make([]byte, chunkSize)
	for {
		n, readErr := io.ReadFull(f, buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			chunks = append(chunks, chunk)
			hasher.Write(chunk)
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return "", nil, readErr
		}
	}
	if len(chunks) == 0 {
		chunks = append(chunks, []byte{})
	}
	return hex.EncodeToString(hasher.Sum(nil)), chunks, nil
}

func isUploadSessionNotFound(err error) bool {
	var he *uploadHTTPError
	if errors.As(err, &he) {
		return he.statusCode == 404 && strings.Contains(he.body, "upload session not found")
	}
	return false
}

func receivedSet(indices []int) map[int]bool {
	out := make(map[int]bool, len(indices))
	for _, i := range indices {
		out[i] = true
	}
	return out
}