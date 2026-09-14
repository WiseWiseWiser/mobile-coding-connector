package client

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveUploadDirTarget resolves the user-supplied remote path against server
// home without applying cp -R nesting. Trailing slashes are trimmed.
func ResolveUploadDirTarget(localDir, remotePath, home string) string {
	localDir = filepath.Clean(localDir)
	baseName := filepath.Base(localDir)
	logical := strings.TrimSpace(remotePath)
	if logical == "" {
		logical = baseName
	}
	logical = strings.TrimSuffix(logical, "/")
	logical = filepath.ToSlash(logical)
	if strings.HasPrefix(logical, "/") {
		return filepath.Clean(logical)
	}
	return filepath.Clean(strings.TrimRight(home, "/") + "/" + logical)
}

// ResolveEffectiveUploadDir applies cp -R destination rules given the remote
// target existence/type:
//
//   - missing target → effective = target (create; copy contents into it)
//   - target is file → error
//   - target is dir  → effective = target/basename(localDir); error if that path is a file
func ResolveEffectiveUploadDir(localDir, remoteTarget string, targetInfo *PathInfo) (effective string, err error) {
	localDir = filepath.Clean(localDir)
	baseName := filepath.Base(localDir)
	remoteTarget = filepath.Clean(remoteTarget)

	if targetInfo == nil || !targetInfo.Exists {
		return remoteTarget, nil
	}
	if !targetInfo.IsDir {
		return "", fmt.Errorf("upload destination %q is a file; refusing directory upload", filepath.ToSlash(remoteTarget))
	}
	effective = filepath.Join(remoteTarget, baseName)
	return effective, nil
}
