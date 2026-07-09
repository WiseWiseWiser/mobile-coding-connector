package agentcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

const downloadHelp = `Usage: remote-agent download <REMOTE_PATH> [LOCAL_PATH]

Download a remote file or directory from the server.

Arguments:
  REMOTE_PATH   Path on the server. May use ~/ for the server's home directory.
  LOCAL_PATH    Destination on this machine. Optional; defaults to the remote
                basename. If LOCAL_PATH ends with '/', the basename is appended.
                For directories, LOCAL_PATH is the mirror root.

Examples:
  remote-agent download '~/server.log'
  remote-agent download /tmp/foo.txt ./foo.txt
  remote-agent download /tmp/foo.txt
  remote-agent download uploads/mirror ./local-mirror
`

func runDownload(cli *client.Client, args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(downloadHelp)
		return nil
	}
	if len(args) < 1 {
		return fmt.Errorf("download requires <REMOTE_PATH> [LOCAL_PATH]; see 'remote-agent download --help'")
	}
	if len(args) > 2 {
		return fmt.Errorf("download takes at most 2 arguments, got %d", len(args))
	}

	remotePath := args[0]
	localPath := ""
	if len(args) == 2 {
		localPath = args[1]
	}

	resolvedRemote, err := cli.ResolveRemoteFilePath(remotePath)
	if err != nil {
		return err
	}

	info, err := cli.CheckPath(resolvedRemote)
	if err != nil {
		return fmt.Errorf("failed to check remote path: %w", err)
	}
	if !info.Exists {
		return fmt.Errorf("remote path %q is missing or does not exist", remotePath)
	}
	if info.IsDir {
		return runDownloadDir(cli, remotePath, localPath)
	}

	fmt.Printf("Downloading %s -> %s\n", remotePath, describeLocal(localPath))

	result, err := cli.DownloadFile(remotePath, localPath, client.DownloadOptions{}, printDownloadProgress)
	if err != nil {
		return err
	}

	fmt.Printf("Download complete: %s (%s)\n", result.LocalPath, formatSize(result.Size))
	return nil
}

func runDownloadDir(cli *client.Client, remotePath, localPath string) error {
	itemCount, _, totalSize, err := cli.CountDownloadDirItems(remotePath)
	if err != nil {
		return err
	}

	logicalLocal := describeDownloadDirLocal(remotePath, localPath)
	fmt.Printf("Downloading %s -> %s (%d items, %s)\n",
		remotePath, logicalLocal, itemCount, formatSize(totalSize))

	result, err := cli.DownloadDir(remotePath, localPath, client.DownloadOptions{}, printDownloadDirProgress)
	if err != nil {
		return err
	}

	if itemCount == result.FileCount {
		fmt.Println()
	}

	summary := fmt.Sprintf("Download complete: %s (%d files, %s",
		logicalLocal, result.FileCount, formatSize(result.TotalSize))
	if result.SkippedCount > 0 || result.ResumedCount > 0 {
		summary += fmt.Sprintf("; %d skipped, %d resumed", result.SkippedCount, result.ResumedCount)
	}
	summary += ")"
	fmt.Println(summary)
	return nil
}

func describeDownloadDirLocal(remotePath, localPath string) string {
	baseName := filepath.Base(strings.TrimSuffix(filepath.ToSlash(remotePath), "/"))
	if localPath == "" {
		return baseName + "/"
	}
	if strings.HasSuffix(localPath, "/") || strings.HasSuffix(localPath, string(os.PathSeparator)) {
		trimmed := strings.TrimSuffix(strings.TrimSuffix(localPath, "/"), string(os.PathSeparator))
		return filepath.ToSlash(filepath.Join(trimmed, baseName)) + "/"
	}
	return filepath.ToSlash(localPath) + "/"
}

func describeLocal(localPath string) string {
	if localPath == "" {
		return "(local basename)"
	}
	return localPath
}