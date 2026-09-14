package agentcli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

const uploadDirStageTotal = 4

type uploadDirStagePrinter struct {
	dryRun     bool
	errW       io.Writer
	current    int // 1-based open stage; 0 = none
	body       *indentingWriter
	uploadOpen bool // true after upload marker; chunk lines go to body
}

func newUploadDirStagePrinter(dryRun bool) *uploadDirStagePrinter {
	return &uploadDirStagePrinter{
		dryRun: dryRun,
		errW:   os.Stderr,
	}
}

func (p *uploadDirStagePrinter) open(n int, kind, msg string) {
	if p.current == n {
		return
	}
	fmt.Fprintf(p.errW, "[%d/%d] %-12s %s\n", n, uploadDirStageTotal, kind, msg)
	pad := strings.Repeat(" ", len(fmt.Sprintf("[%d/%d] ", n, uploadDirStageTotal)))
	p.body = newIndentingWriter(p.errW, pad)
	p.current = n
	p.uploadOpen = n == 3
}

func (p *uploadDirStagePrinter) detail(format string, args ...any) {
	if p.body == nil {
		return
	}
	fmt.Fprintf(p.body, format+"\n", args...)
}

func (p *uploadDirStagePrinter) OnProgress(ev client.UploadDirProgress) {
	switch ev.Phase {
	case client.UploadDirPhaseResolved:
		p.open(1, "resolve", ev.RelativePath)
		switch {
		case ev.TotalItems > 0 && ev.TotalBytes > 0:
			p.detail("notice: %d items, %s", ev.TotalItems, formatSize(ev.TotalBytes))
		case ev.TotalItems > 0:
			p.detail("notice: %d items", ev.TotalItems)
		case ev.TotalBytes > 0:
			p.detail("notice: %s", formatSize(ev.TotalBytes))
		}
	case client.UploadDirPhasePacking:
		p.open(2, "pack", "tar.xz")
		if p.dryRun {
			p.detail("would: pack tar.xz")
			return
		}
		if ev.FileSize > 0 {
			p.detail("notice: archive %s", formatSize(ev.FileSize))
		}
	case client.UploadDirPhaseUploading:
		if p.dryRun {
			p.open(3, "upload", "archive")
			p.detail("would: upload archive")
			return
		}
		sizeMsg := formatSize(ev.FileSize)
		if ev.FileSize == 0 && ev.TotalBytes > 0 {
			sizeMsg = formatSize(ev.TotalBytes)
		}
		p.open(3, "upload", sizeMsg)
		if ev.Chunk.TotalChunks > 0 || ev.Chunk.Phase != "" {
			p.printChunk(ev.Chunk)
		}
	case client.UploadDirPhaseApplying:
		dest := ev.RelativePath
		if dest == "" {
			dest = "extract/merge"
		}
		p.open(4, "apply", dest)
		if p.dryRun {
			p.detail("would: apply extract/merge")
		}
	default:
		if ev.Chunk.TotalChunks > 0 || ev.Chunk.Phase != "" {
			if p.current != 3 {
				p.open(3, "upload", formatSize(ev.FileSize))
			}
			p.printChunk(ev.Chunk)
		}
	}
}

func (p *uploadDirStagePrinter) printChunk(chunk client.UploadProgress) {
	if p.body == nil {
		return
	}
	switch chunk.Phase {
	case client.UploadChunkRetrying:
		p.detail("chunk %d/%d retrying (attempt %d/%d: %s)...",
			chunk.ChunkIndex+1, chunk.TotalChunks,
			chunk.Attempt, chunk.MaxAttempts,
			shortUploadErr(chunk.Err))
	case client.UploadChunkSkipped:
		percent := chunkPercent(chunk.CompletedBytes, chunk.TotalBytes)
		p.detail("chunk %d/%d skipped (cached, %s / %s, %d%%)",
			chunk.ChunkIndex+1, chunk.TotalChunks,
			formatSize(chunk.CompletedBytes), formatSize(chunk.TotalBytes), percent)
	case client.UploadChunkUploaded:
		percent := chunkPercent(chunk.CompletedBytes, chunk.TotalBytes)
		suffix := ""
		if chunk.Attempt > 1 {
			suffix = fmt.Sprintf(", %d attempts", chunk.Attempt)
		}
		p.detail("chunk %d/%d uploaded (%s / %s, %d%%%s)",
			chunk.ChunkIndex+1, chunk.TotalChunks,
			formatSize(chunk.CompletedBytes), formatSize(chunk.TotalBytes), percent, suffix)
	default:
		percent := chunkPercent(chunk.CompletedBytes, chunk.TotalBytes)
		p.detail("chunk %d/%d uploaded (%s / %s, %d%%)",
			chunk.ChunkIndex+1, chunk.TotalChunks,
			formatSize(chunk.CompletedBytes), formatSize(chunk.TotalBytes), percent)
	}
}

func (p *uploadDirStagePrinter) FinishApply(overridden []string) {
	if p.current != 4 {
		p.open(4, "apply", "extract/merge")
	}
	if len(overridden) > 0 {
		const maxShow = 20
		p.detail("warning: overridden %d existing file(s)", len(overridden))
		show := overridden
		if len(show) > maxShow {
			show = overridden[:maxShow]
		}
		for _, path := range show {
			p.detail("warning:   %s", path)
		}
		if len(overridden) > maxShow {
			p.detail("warning:   … and %d more", len(overridden)-maxShow)
		}
	}
	if !p.dryRun {
		p.detail("ok")
	}
}

func (p *uploadDirStagePrinter) PrintProduct(result *client.UploadDirResult) {
	fmt.Fprintln(p.errW) // blank separator on stderr
	if p.dryRun {
		fmt.Printf("would: upload %s (%d files, %s)\n",
			result.Path, result.FileCount, formatSize(result.TotalSize))
		return
	}
	extra := ""
	if result.ArchiveSize > 0 {
		extra = fmt.Sprintf(", archive %s", formatSize(result.ArchiveSize))
	}
	fmt.Printf("uploaded %s (%d files, %s%s)\n",
		result.Path, result.FileCount, formatSize(result.TotalSize), extra)
}
