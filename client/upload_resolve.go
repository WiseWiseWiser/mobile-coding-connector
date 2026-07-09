package client

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveRemoteFilePath applies single-file upload destination rules using home
// as the server's home directory.
func ResolveRemoteFilePath(localFile, remotePath, home string) string {
	baseName := filepath.Base(localFile)
	if remotePath == "" {
		remotePath = baseName
	} else if strings.HasSuffix(remotePath, "/") {
		remotePath = remotePath + baseName
	}
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = strings.TrimRight(home, "/") + "/" + remotePath
	}
	return remotePath
}

// ResolveRemoteDirPath applies directory-upload destination rules. It returns
// the logical destination (as passed or derived from basename rules) and the
// absolute path on the server.
func ResolveRemoteDirPath(localDir, remotePath, home string) (logical string, absolute string) {
	baseName := filepath.Base(localDir)
	logical = remotePath
	if logical == "" {
		logical = baseName
	} else if strings.HasSuffix(logical, "/") {
		logical = strings.TrimSuffix(logical, "/") + "/" + baseName
	}
	logical = filepath.ToSlash(logical)
	if strings.HasPrefix(logical, "/") {
		return logical, filepath.Clean(logical)
	}
	absolute = strings.TrimRight(home, "/") + "/" + logical
	return logical, filepath.Clean(absolute)
}

func (c *Client) resolveRemoteDir(localDir, remotePath string) (logical string, absolute string, err error) {
	home, err := c.GetHome()
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve server home dir: %w", err)
	}
	logical, absolute = ResolveRemoteDirPath(localDir, remotePath, home.Home)
	return logical, absolute, nil
}