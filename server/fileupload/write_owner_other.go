//go:build !unix

package fileupload

import "os"

// preserveOwner is a no-op on platforms without uid/gid ownership.
func preserveOwner(path string, info os.FileInfo) {}
