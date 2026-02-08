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

export interface UploadProgress {
    loaded: number;
    total: number;
    percent: number;
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

export function uploadFile(
    file: File,
    destPath: string,
    onProgress?: (progress: UploadProgress) => void,
): Promise<UploadResult> {
    return new Promise((resolve, reject) => {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('path', destPath);

        const xhr = new XMLHttpRequest();

        xhr.upload.addEventListener('progress', (e) => {
            if (e.lengthComputable && onProgress) {
                onProgress({
                    loaded: e.loaded,
                    total: e.total,
                    percent: Math.round((e.loaded / e.total) * 100),
                });
            }
        });

        xhr.addEventListener('load', () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                try {
                    resolve(JSON.parse(xhr.responseText));
                } catch {
                    reject(new Error('Invalid response'));
                }
            } else {
                try {
                    const data = JSON.parse(xhr.responseText);
                    reject(new Error(data.error || 'Failed to upload file'));
                } catch {
                    reject(new Error('Failed to upload file'));
                }
            }
        });

        xhr.addEventListener('error', () => {
            reject(new Error('Network error during upload'));
        });

        xhr.addEventListener('abort', () => {
            reject(new Error('Upload aborted'));
        });

        xhr.open('POST', '/api/files/upload');
        xhr.send(formData);
    });
}
