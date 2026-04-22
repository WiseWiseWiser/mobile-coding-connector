package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

func runUpload(cli *client.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("upload requires <LOCAL_FILE> [REMOTE_PATH]")
	}
	if len(args) > 2 {
		return fmt.Errorf("upload takes at most 2 arguments, got %d", len(args))
	}

	localFile := args[0]
	remotePath := ""
	if len(args) == 2 {
		remotePath = args[1]
	}

	stat, err := os.Stat(localFile)
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}
	chmodExec := isExecutableMode(stat.Mode())

	fmt.Printf("Uploading %s (%s) -> %s\n", localFile, formatSize(stat.Size()), describeRemote(remotePath))

	result, err := cli.UploadFile(localFile, remotePath, client.UploadOptions{
		ChmodExec: chmodExec,
	}, func(p client.UploadProgress) {
		percent := 100
		if p.TotalBytes > 0 {
			percent = int(p.CompletedBytes * 100 / p.TotalBytes)
		}
		fmt.Printf("  chunk %d/%d uploaded (%s / %s, %d%%)\n",
			p.ChunkIndex+1, p.TotalChunks,
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent)
	})
	if err != nil {
		return err
	}

	fmt.Printf("Upload complete: %s (%s)\n", result.Path, formatSize(result.Size))
	return nil
}

func isExecutableMode(mode os.FileMode) bool {
	return mode.IsRegular() && mode&0o111 != 0
}

func describeRemote(remotePath string) string {
	if remotePath == "" {
		return "(server home dir)"
	}
	return remotePath
}

func formatSize(n int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.2f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.2f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.2f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
