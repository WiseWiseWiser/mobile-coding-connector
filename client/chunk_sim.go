package client

// ChunkCount returns the number of chunks for a file of the given size.
func ChunkCount(size int64, chunkSize int) int {
	if size == 0 {
		return 1
	}
	return int((size + int64(chunkSize) - 1) / int64(chunkSize))
}

// ChunkLenAt returns the byte length of chunk index for a file of totalSize.
func ChunkLenAt(index int, totalSize int64, chunkSize int) int64 {
	start := int64(index) * int64(chunkSize)
	if start >= totalSize {
		return 0
	}
	end := start + int64(chunkSize)
	if end > totalSize {
		return totalSize - start
	}
	return int64(chunkSize)
}

// SimulateUploadChunks emits UploadProgress events for a dry-run upload plan.
func SimulateUploadChunks(totalSize int64, chunkSize int, onProgress func(UploadProgress)) {
	if onProgress == nil {
		return
	}
	totalChunks := ChunkCount(totalSize, chunkSize)
	completedBytes := int64(0)
	for i := 0; i < totalChunks; i++ {
		completedBytes += ChunkLenAt(i, totalSize, chunkSize)
		onProgress(UploadProgress{
			ChunkIndex:     i,
			TotalChunks:    totalChunks,
			CompletedBytes: completedBytes,
			TotalBytes:     totalSize,
			Phase:          UploadChunkUploaded,
		})
	}
}

// SimulateDownloadChunks emits DownloadProgress events for a dry-run download plan.
func SimulateDownloadChunks(totalSize, startOffset int64, onProgress func(DownloadProgress)) {
	if onProgress == nil {
		return
	}
	remaining := totalSize - startOffset
	if remaining < 0 {
		remaining = 0
	}
	totalChunks := ChunkCount(remaining, ChunkSize)
	completedBytes := startOffset
	for i := 0; i < totalChunks; i++ {
		completedBytes += ChunkLenAt(i, remaining, ChunkSize)
		onProgress(DownloadProgress{
			CompletedBytes: completedBytes,
			TotalBytes:     totalSize,
			Phase:          DownloadPhaseDownloading,
		})
	}
}