package main

import (
	"fmt"

	"github.com/xhd2015/ai-critic/client"
)

const downloadHelp = `Usage: remote-agent download <REMOTE_PATH> [LOCAL_PATH]

Download a remote file from the server.

Arguments:
  REMOTE_PATH   Path on the server. May use ~/ for the server's home directory.
  LOCAL_PATH    Destination on this machine. Optional; defaults to the remote
                file's basename.

Examples:
  remote-agent download '~/server.log'
  remote-agent download /tmp/foo.txt ./foo.txt
  remote-agent download /tmp/foo.txt
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