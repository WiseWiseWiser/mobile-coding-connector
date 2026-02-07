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

export async function uploadFile(file: File, destPath: string): Promise<UploadResult> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('path', destPath);

    const resp = await fetch('/api/files/upload', {
        method: 'POST',
        body: formData,
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to upload file');
    }
    return resp.json();
}
