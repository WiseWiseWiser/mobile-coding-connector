// File Transfer inbox API client

export interface FileTransferEntry {
    name: string;
    size: number;
    uploaded_at: string;
}

export interface FileTransferListResult {
    files: FileTransferEntry[];
}

export interface FileTransferUploadResult {
    id: string;
    name: string;
    size: number;
    uploaded_at: string;
}

export async function listFileTransfer(): Promise<FileTransferEntry[]> {
    const resp = await fetch('/api/file-transfer');
    if (!resp.ok) {
        const err = await resp.json().catch(() => ({}));
        throw new Error((err as { error?: string }).error || `Failed to list files (${resp.status})`);
    }
    const data: FileTransferListResult = await resp.json();
    return data.files ?? [];
}

export async function uploadFileTransfer(file: File): Promise<FileTransferUploadResult> {
    const form = new FormData();
    form.append('file', file);
    const resp = await fetch('/api/file-transfer/upload', {
        method: 'POST',
        body: form,
    });
    if (!resp.ok) {
        const err = await resp.json().catch(() => ({}));
        throw new Error((err as { error?: string }).error || `Upload failed (${resp.status})`);
    }
    return resp.json();
}

export function fileTransferDownloadUrl(name: string): string {
    const params = new URLSearchParams({ name });
    return `/api/file-transfer/download?${params}`;
}

export async function deleteFileTransfer(name: string): Promise<void> {
    const resp = await fetch('/api/file-transfer', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
    });
    if (!resp.ok) {
        const err = await resp.json().catch(() => ({}));
        throw new Error((err as { error?: string }).error || `Delete failed (${resp.status})`);
    }
}