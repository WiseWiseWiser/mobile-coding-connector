// File upload API client

export interface ServerFileInfo {
    exists: boolean;
    path: string;
    size: number;
    mod_time?: string;
    is_dir: boolean;
    file_mode?: string;
}

export interface UploadResult {
    status: string;
    path: string;
    size: number;
    original_name: string;
}

const UploadPhases = {
    Uploading: 'uploading',
    Merging: 'merging',
} as const;

type UploadPhase = typeof UploadPhases[keyof typeof UploadPhases];

export { UploadPhases };
export type { UploadPhase };

export interface UploadProgress {
    loaded: number;
    total: number;
    percent: number;
    /** Current upload phase */
    phase: UploadPhase;
    /** Current chunk index (0-based) */
    chunkIndex?: number;
    /** Total number of chunks */
    totalChunks?: number;
    /** Bytes loaded within the current chunk */
    chunkLoaded?: number;
    /** Total bytes in the current chunk */
    chunkTotal?: number;
}

export async function checkServerFile(path: string): Promise<ServerFileInfo> {
    const resp = await fetch('/api/files/check', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path }),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to check file');
    }
    return resp.json();
}

const CHUNK_SIZE = 2 * 1024 * 1024; // 2MB per chunk

/** Upload a single chunk using XHR for progress tracking */
function uploadChunkWithProgress(
    uploadId: string,
    chunkIndex: number,
    chunk: Blob,
    onChunkProgress: (loaded: number, total: number) => void,
): Promise<void> {
    return new Promise((resolve, reject) => {
        const formData = new FormData();
        formData.append('upload_id', uploadId);
        formData.append('chunk_index', String(chunkIndex));
        formData.append('chunk', chunk, `chunk_${chunkIndex}`);

        const xhr = new XMLHttpRequest();

        xhr.upload.addEventListener('progress', (e) => {
            if (e.lengthComputable) {
                onChunkProgress(e.loaded, e.total);
            }
        });

        xhr.addEventListener('load', () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                resolve();
            } else {
                try {
                    const data = JSON.parse(xhr.responseText);
                    reject(new Error(data.error || `Failed to upload chunk ${chunkIndex}`));
                } catch {
                    reject(new Error(`Failed to upload chunk ${chunkIndex}`));
                }
            }
        });

        xhr.addEventListener('error', () => {
            reject(new Error(`Network error uploading chunk ${chunkIndex}`));
        });

        xhr.addEventListener('abort', () => {
            reject(new Error(`Upload chunk ${chunkIndex} aborted`));
        });

        xhr.open('POST', '/api/files/upload/chunk');
        xhr.send(formData);
    });
}

export async function uploadFile(
    file: File,
    destPath: string,
    onProgress?: (progress: UploadProgress) => void,
): Promise<UploadResult> {
    const totalSize = file.size;
    const totalChunks = Math.max(1, Math.ceil(totalSize / CHUNK_SIZE));

    // Step 1: Initialize chunked upload session
    const initResp = await fetch('/api/files/upload/init', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            path: destPath,
            total_chunks: totalChunks,
            total_size: totalSize,
        }),
    });
    if (!initResp.ok) {
        const data = await initResp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to initialize upload');
    }
    const { upload_id } = await initResp.json();

    // Step 2: Upload chunks sequentially with per-chunk progress
    let completedChunkBytes = 0;

    for (let i = 0; i < totalChunks; i++) {
        const start = i * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, totalSize);
        const chunkSize = end - start;
        const chunk = file.slice(start, end);

        const baseBytes = completedChunkBytes;
        await uploadChunkWithProgress(upload_id, i, chunk, (loaded, total) => {
            onProgress?.({
                loaded: baseBytes + loaded,
                total: totalSize,
                percent: Math.round(((baseBytes + loaded) / totalSize) * 100),
                phase: UploadPhases.Uploading,
                chunkIndex: i,
                totalChunks,
                chunkLoaded: loaded,
                chunkTotal: total,
            });
        });

        completedChunkBytes += chunkSize;
    }

    // Step 3: Merging phase
    onProgress?.({
        loaded: totalSize,
        total: totalSize,
        percent: 100,
        phase: UploadPhases.Merging,
        chunkIndex: totalChunks - 1,
        totalChunks,
    });

    const completeResp = await fetch('/api/files/upload/complete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ upload_id }),
    });
    if (!completeResp.ok) {
        const data = await completeResp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to complete upload');
    }

    const result = await completeResp.json();
    return {
        status: result.status,
        path: result.path,
        size: result.size,
        original_name: file.name,
    };
}
