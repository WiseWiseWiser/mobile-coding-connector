package agentcli

import (
	"fmt"
	"os"

	"github.com/xhd2015/ai-critic/client"
)

const uploadHelp = `Usage: remote-agent upload <LOCAL_FILE> [REMOTE_PATH]

Upload a local file to the server using chunked upload.

Arguments:
  LOCAL_FILE    Path to the file on this machine.
  REMOTE_PATH   Destination path on the server. Optional; defaults to the
                file's basename. If REMOTE_PATH ends with '/', the basename
                is appended.

Examples:
  remote-agent upload ./foo.txt /tmp/foo.txt
  remote-agent upload ./foo.txt /tmp/          # basename appended
  remote-agent upload ./foo.txt                # uses saved config + basename
`

func runUpload(cli *client.Client, args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(uploadHelp)
		return nil
	}
	if len(args) < 1 {
		return fmt.Errorf("upload requires <LOCAL_FILE> [REMOTE_PATH]; see 'remote-agent upload --help'")
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
	}, printUploadProgress)
	if err != nil {
		if hint := uploadFailureHint(err); hint != "" {
			return fmt.Errorf("%w\n  %s", err, hint)
		}
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
