package agentcli

import (
	"fmt"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

func printDownloadDryRunProgress(p client.DownloadProgress) {
	percent := downloadPercent(p.CompletedBytes, p.TotalBytes)
	totalChunks := client.ChunkCount(p.TotalBytes, client.ChunkSize)
	chunkIndex := client.ChunkCount(p.CompletedBytes, client.ChunkSize)
	if p.CompletedBytes > 0 && p.CompletedBytes%int64(client.ChunkSize) == 0 && p.CompletedBytes < p.TotalBytes {
		// keep chunkIndex as computed
	} else if p.CompletedBytes == p.TotalBytes && p.TotalBytes > 0 {
		chunkIndex = totalChunks
	}
	fmt.Printf("  would download chunk %d/%d (%s / %s, %d%%)\n",
		chunkIndex, totalChunks,
		formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent)
}

func printDownloadProgress(p client.DownloadProgress) {
	switch p.Phase {
	case client.DownloadPhaseRetrying:
		fmt.Printf("  retrying (attempt %d/%d: %s)...\n",
			p.Attempt, p.MaxAttempts,
			shortDownloadErr(p.Err))
	default:
		percent := downloadPercent(p.CompletedBytes, p.TotalBytes)
		fmt.Printf("  downloaded %s / %s (%d%%)\n",
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent)
	}
}

func printDownloadDirDryRunProgress(p client.DownloadDirProgress) {
	overall := formatOverallPercent(p.CompletedBytes, p.TotalBytes)
	switch p.Phase {
	case client.DownloadDirPhaseFileStart:
		fmt.Printf("  [%d/%d] %s (%s) — %s\n",
			p.FileIndex, p.TotalItems,
			p.RelativePath, formatSize(p.FileTotal),
			overall)
	case client.DownloadDirPhaseDirCreated:
		fmt.Printf("  [%d/%d] would create %s — %s\n",
			p.FileIndex, p.TotalItems,
			p.RelativePath,
			overall)
	case client.DownloadDirPhaseDirExists:
		fmt.Printf("  [%d/%d] %s — %s\n",
			p.FileIndex, p.TotalItems,
			p.RelativePath,
			overall)
		fmt.Printf("    would skip (already exists) — %s\n", overall)
	case client.DownloadDirPhaseSkipped:
		fmt.Printf("    would skip (already complete, %s / %s) — %s\n",
			formatSize(p.FileTotal), formatSize(p.FileTotal),
			overall)
	case client.DownloadDirPhaseResumed:
		percent := downloadPercent(p.FileCompleted, p.FileTotal)
		fmt.Printf("    would resume at %s / %s (%d%%) — %s\n",
			formatSize(p.FileCompleted), formatSize(p.FileTotal), percent,
			overall)
	case client.DownloadDirPhaseDownloading:
		fmt.Printf("    would download chunk ... — %s\n", overall)
	}
}

func printDownloadDirProgress(p client.DownloadDirProgress) {
	overall := formatOverallPercent(p.CompletedBytes, p.TotalBytes)
	switch p.Phase {
	case client.DownloadDirPhaseFileStart:
		fmt.Printf("  [%d/%d] %s (%s) — %s\n",
			p.FileIndex, p.TotalItems,
			p.RelativePath, formatSize(p.FileTotal),
			overall)
	case client.DownloadDirPhaseDirCreated:
		fmt.Printf("  [%d/%d] created %s — %s\n",
			p.FileIndex, p.TotalItems,
			p.RelativePath,
			overall)
	case client.DownloadDirPhaseDirExists:
		fmt.Printf("  [%d/%d] %s — %s\n",
			p.FileIndex, p.TotalItems,
			p.RelativePath,
			overall)
		fmt.Printf("    skipped (already exists) — %s\n", overall)
	case client.DownloadDirPhaseSkipped:
		fmt.Printf("    skipped (already complete, %s / %s) — %s\n",
			formatSize(p.FileTotal), formatSize(p.FileTotal),
			overall)
	case client.DownloadDirPhaseResumed:
		percent := downloadPercent(p.FileCompleted, p.FileTotal)
		fmt.Printf("    resumed at %s / %s (%d%%) — %s\n",
			formatSize(p.FileCompleted), formatSize(p.FileTotal), percent,
			overall)
	case client.DownloadDirPhaseRetrying:
		fmt.Printf("    retrying (attempt %d/%d: %s)... — %s\n",
			p.Attempt, p.MaxAttempts,
			shortDownloadErr(p.Err),
			overall)
	case client.DownloadDirPhaseDownloading:
		percent := downloadPercent(p.FileCompleted, p.FileTotal)
		fmt.Printf("    downloaded %s / %s (%d%%) — %s\n",
			formatSize(p.FileCompleted), formatSize(p.FileTotal), percent,
			overall)
	}
}

func shortDownloadErr(err error) string {
	if err == nil {
		return "unknown error"
	}
	msg := err.Error()
	if idx := strings.Index(msg, ": "); idx >= 0 {
		return msg[idx+2:]
	}
	return msg
}

func downloadPercent(completed, total int64) int {
	if total == 0 {
		return 100
	}
	return int(completed * 100 / total)
}