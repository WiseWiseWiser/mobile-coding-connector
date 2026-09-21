package agentcli

import (
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

func printUploadDryRunProgress(p client.UploadProgress) {
	switch p.Phase {
	case client.UploadStreamStart:
		printUploadStart(p, true)
	case client.UploadStreamProgress:
		fmt.Printf("  would upload %s / %s\n", formatSize(p.CompletedBytes), formatSize(p.TotalBytes))
	default:
		percent := chunkPercent(p.CompletedBytes, p.TotalBytes)
		fmt.Printf("  would upload chunk %d/%d (%s / %s, %d%%)\n",
			p.ChunkIndex+1, p.TotalChunks,
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent)
	}
}

func printUploadProgress(p client.UploadProgress) {
	switch p.Phase {
	case client.UploadStreamStart:
		printUploadStart(p, false)
	case client.UploadStreamProgress:
		rate := ""
		if p.BytesPerSec > 0 {
			rate = "   " + formatSize(p.BytesPerSec) + "/s"
		}
		fmt.Printf("  %s / %s%s\n", formatSize(p.CompletedBytes), formatSize(p.TotalBytes), rate)
	case client.UploadStreamResuming:
		msg := fmt.Sprintf("warning: websocket lost at %s; resuming from offset %d",
			formatSize(p.CompletedBytes), p.CompletedBytes)
		if stderrColorEnabled() {
			msg = colorLabel(msg)
		}
		fmt.Fprintln(os.Stderr, msg)
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

func printUploadStart(p client.UploadProgress, dryRun bool) {
	prefix := "  "
	if dryRun {
		prefix = "  would: "
	}
	if p.OrigBytes > 0 && p.TotalBytes > 0 && p.TotalBytes < p.OrigBytes {
		fmt.Printf("%sgzip %s → %s wire\n", prefix, formatSize(p.OrigBytes), formatSize(p.TotalBytes))
		return
	}
	if p.TotalBytes > 0 {
		fmt.Printf("%s%s wire\n", prefix, formatSize(p.TotalBytes))
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
		return "hint: server lost the upload session (restart or timeout) — re-run upload; websocket uploads resume from the last acked offset"
	}
	if strings.Contains(msg, "hash mismatch") {
		return "hint: remote cache is for a different payload — re-run upload (a new hash starts from offset 0)"
	}
	return ""
}
