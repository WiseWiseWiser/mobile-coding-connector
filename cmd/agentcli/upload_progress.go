package agentcli

import (
	"fmt"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

func printUploadProgress(p client.UploadProgress) {
	switch p.Phase {
	case client.UploadChunkRetrying:
		fmt.Printf("  chunk %d/%d retrying (attempt %d/%d: %s)...\n",
			p.ChunkIndex+1, p.TotalChunks,
			p.Attempt, p.MaxAttempts,
			shortUploadErr(p.Err))
	case client.UploadChunkSkipped:
		percent := 100
		if p.TotalBytes > 0 {
			percent = int(p.CompletedBytes * 100 / p.TotalBytes)
		}
		fmt.Printf("  chunk %d/%d skipped (cached, %s / %s, %d%%)\n",
			p.ChunkIndex+1, p.TotalChunks,
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent)
	case client.UploadChunkUploaded:
		percent := 100
		if p.TotalBytes > 0 {
			percent = int(p.CompletedBytes * 100 / p.TotalBytes)
		}
		suffix := ""
		if p.Attempt > 1 {
			suffix = fmt.Sprintf(", %d attempts", p.Attempt)
		}
		fmt.Printf("  chunk %d/%d uploaded (%s / %s, %d%%%s)\n",
			p.ChunkIndex+1, p.TotalChunks,
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent, suffix)
	default:
		percent := 100
		if p.TotalBytes > 0 {
			percent = int(p.CompletedBytes * 100 / p.TotalBytes)
		}
		fmt.Printf("  chunk %d/%d uploaded (%s / %s, %d%%)\n",
			p.ChunkIndex+1, p.TotalChunks,
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent)
	}
}

func shortUploadErr(err error) string {
	if err == nil {
		return "unknown error"
	}
	msg := err.Error()
	if idx := strings.Index(msg, ": "); idx >= 0 {
		return msg[idx+2:]
	}
	return msg
}

func printUploadDirProgress(p client.UploadDirProgress) {
	overall := formatOverallPercent(p.CompletedBytes, p.TotalBytes)
	switch p.Phase {
	case client.UploadDirPhaseFileStart:
		fmt.Printf("  [%d/%d] %s (%s) — %s\n",
			p.FileIndex, p.TotalItems,
			p.RelativePath, formatSize(p.FileSize),
			overall)
	case client.UploadDirPhaseDirCreated:
		fmt.Printf("  [%d/%d] created %s — %s\n",
			p.FileIndex, p.TotalItems,
			p.RelativePath,
			overall)
	default:
		printUploadDirChunkProgress(p.Chunk, overall)
	}
}

func printUploadDirChunkProgress(p client.UploadProgress, overall string) {
	switch p.Phase {
	case client.UploadChunkRetrying:
		fmt.Printf("    chunk %d/%d retrying (attempt %d/%d: %s)... — %s\n",
			p.ChunkIndex+1, p.TotalChunks,
			p.Attempt, p.MaxAttempts,
			shortUploadErr(p.Err),
			overall)
	case client.UploadChunkSkipped:
		percent := chunkPercent(p.CompletedBytes, p.TotalBytes)
		fmt.Printf("    chunk %d/%d skipped (cached, %s / %s, %d%%) — %s\n",
			p.ChunkIndex+1, p.TotalChunks,
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent,
			overall)
	case client.UploadChunkUploaded:
		percent := chunkPercent(p.CompletedBytes, p.TotalBytes)
		suffix := ""
		if p.Attempt > 1 {
			suffix = fmt.Sprintf(", %d attempts", p.Attempt)
		}
		fmt.Printf("    chunk %d/%d uploaded (%s / %s, %d%%%s) — %s\n",
			p.ChunkIndex+1, p.TotalChunks,
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent, suffix,
			overall)
	default:
		percent := chunkPercent(p.CompletedBytes, p.TotalBytes)
		fmt.Printf("    chunk %d/%d uploaded (%s / %s, %d%%) — %s\n",
			p.ChunkIndex+1, p.TotalChunks,
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent,
			overall)
	}
}

func formatOverallPercent(completed, total int64) string {
	percent := 100
	if total > 0 {
		percent = int(completed * 100 / total)
	}
	return fmt.Sprintf("%d%% overall", percent)
}

func chunkPercent(completed, total int64) int {
	if total == 0 {
		return 100
	}
	return int(completed * 100 / total)
}

func uploadFailureHint(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.Contains(msg, "upload session not found") {
		return "hint: server lost the upload session (restart or timeout) — re-run upload from start; completed chunks are not preserved across sessions"
	}
	return ""
}