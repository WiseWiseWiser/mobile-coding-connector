// File download/browse API client

export interface BrowseEntry {
    name: string;
    path: string;
    is_dir: boolean;
    size: number;
}

export interface BrowseResult {
    path: string;
    entries: BrowseEntry[];
}

export async function browseDirectory(path: string): Promise<BrowseResult> {
    const params = new URLSearchParams({ path });
    const resp = await fetch(`/api/files/browse?${params}`);
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to browse directory');
    }
    return resp.json();
}

export function getDownloadUrl(path: string): string {
    const params = new URLSearchParams({ path });
    return `/api/files/download?${params}`;
}

export interface DownloadProgress {
    loaded: number;
    total: number;
    percent: number;
}

export function downloadFile(
    path: string,
    onProgress?: (progress: DownloadProgress) => void,
): Promise<void> {
    return new Promise((resolve, reject) => {
        const url = getDownloadUrl(path);
        const xhr = new XMLHttpRequest();

        xhr.addEventListener('progress', (e) => {
            if (onProgress) {
                const total = e.lengthComputable ? e.total : 0;
                const percent = total > 0 ? Math.round((e.loaded / total) * 100) : 0;
                onProgress({ loaded: e.loaded, total, percent });
            }
        });

        xhr.addEventListener('load', () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                // Create a blob and trigger download
                const blob = new Blob([xhr.response]);
                const blobUrl = URL.createObjectURL(blob);
                const filename = path.split('/').pop() || 'download';
                const a = document.createElement('a');
                a.href = blobUrl;
                a.download = filename;
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(blobUrl);
                resolve();
            } else {
                reject(new Error('Download failed'));
            }
        });

        xhr.addEventListener('error', () => {
            reject(new Error('Network error during download'));
        });

        xhr.addEventListener('abort', () => {
            reject(new Error('Download aborted'));
        });

        xhr.open('GET', url);
        xhr.responseType = 'arraybuffer';
        xhr.send();
    });
}
