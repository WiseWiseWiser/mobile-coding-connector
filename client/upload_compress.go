package client

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"time"
)

// gzipBytesDeterministic produces a stable gzip stream for the same input so
// file_hash-based upload resume keeps working across retries/processes.
// ModTime is forced to zero; Name/Comment stay empty.
func gzipBytesDeterministic(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}
	zw.Header.ModTime = time.Time{}
	zw.Header.Name = ""
	zw.Header.Comment = ""
	if _, err := zw.Write(src); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// prepareUploadPayload reads localPath and optionally gzip-compresses it.
// When noCompress is false, gzip is applied unless the compressed size is not
// smaller than the original (then raw bytes are uploaded).
func prepareUploadPayload(localPath string, noCompress bool) (payload []byte, compressed bool, err error) {
	raw, err := os.ReadFile(localPath)
	if err != nil {
		return nil, false, err
	}
	if noCompress {
		return raw, false, nil
	}
	gz, err := gzipBytesDeterministic(raw)
	if err != nil {
		return nil, false, err
	}
	if len(gz) >= len(raw) {
		return raw, false, nil
	}
	return gz, true, nil
}

func computeBytesChunkPlan(data []byte, chunkSize int) (fileHash string, chunks [][]byte) {
	if chunkSize <= 0 {
		chunkSize = ChunkSize
	}
	hasher := sha256.New()
	if len(data) == 0 {
		chunks = [][]byte{{}}
		hasher.Write(nil)
		return hex.EncodeToString(hasher.Sum(nil)), chunks
	}
	for off := 0; off < len(data); off += chunkSize {
		end := off + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := append([]byte(nil), data[off:end]...)
		chunks = append(chunks, chunk)
		hasher.Write(chunk)
	}
	return hex.EncodeToString(hasher.Sum(nil)), chunks
}

