package machinebackup

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const archiveSessionTTL = 30 * time.Minute

type archiveSession struct {
	path    string
	created time.Time
}

var archiveSessions sync.Map

func registerArchiveSession(path string) (string, error) {
	token, err := newArchiveToken()
	if err != nil {
		return "", err
	}
	archiveSessions.Store(token, &archiveSession{path: path, created: time.Now().UTC()})
	go purgeExpiredArchiveSessions()
	return token, nil
}

func openArchiveSession(token string) (io.ReadCloser, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("archive token is required")
	}
	v, ok := archiveSessions.LoadAndDelete(token)
	if !ok {
		return nil, fmt.Errorf("archive session not found or expired")
	}
	sess, ok := v.(*archiveSession)
	if !ok || sess == nil {
		return nil, fmt.Errorf("invalid archive session")
	}
	if time.Since(sess.created) > archiveSessionTTL {
		os.Remove(sess.path)
		return nil, fmt.Errorf("archive session expired")
	}
	f, err := os.Open(sess.path)
	if err != nil {
		os.Remove(sess.path)
		return nil, fmt.Errorf("open archive: %w", err)
	}
	return &archiveSessionFile{File: f, path: sess.path}, nil
}

type archiveSessionFile struct {
	*os.File
	path string
}

func (f *archiveSessionFile) Close() error {
	err := f.File.Close()
	os.Remove(f.path)
	return err
}

func newArchiveToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate archive token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func purgeExpiredArchiveSessions() {
	cutoff := time.Now().UTC().Add(-archiveSessionTTL)
	archiveSessions.Range(func(key, value any) bool {
		sess, ok := value.(*archiveSession)
		if !ok || sess == nil {
			archiveSessions.Delete(key)
			return true
		}
		if sess.created.Before(cutoff) {
			os.Remove(sess.path)
			archiveSessions.Delete(key)
		}
		return true
	})
}

