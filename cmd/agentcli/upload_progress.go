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