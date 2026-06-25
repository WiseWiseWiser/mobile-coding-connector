package main

import (
	"fmt"

	"github.com/xhd2015/ai-critic/client"
)

func runDownload(cli *client.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("download requires <REMOTE_PATH> [LOCAL_PATH]")
	}
	if len(args) > 2 {
		return fmt.Errorf("download takes at most 2 arguments, got %d", len(args))
	}

	remotePath := args[0]
	localPath := ""
	if len(args) == 2 {
		localPath = args[1]
	}

	fmt.Printf("Downloading %s -> %s\n", remotePath, describeLocal(localPath))

	result, err := cli.DownloadFile(remotePath, localPath, func(p client.DownloadProgress) {
		percent := 100
		if p.TotalBytes > 0 {
			percent = int(p.CompletedBytes * 100 / p.TotalBytes)
		}
		fmt.Printf("  downloaded %s / %s (%d%%)\n",
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent)
	})
	if err != nil {
		return err
	}

	fmt.Printf("Download complete: %s (%s)\n", result.LocalPath, formatSize(result.Size))
	return nil
}

func describeLocal(localPath string) string {
	if localPath == "" {
		return "(local basename)"
	}
	return localPath
}