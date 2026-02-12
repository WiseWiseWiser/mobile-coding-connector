// File API for server file management

export interface FilePartialResult {
    content: string;
    totalSize: number;
    offset: number;
    hasMore: boolean;
}

export async function fetchFilePartial(
    filePath: string,
    offset: number = 0,
    limit: number = 8192
): Promise<FilePartialResult> {
    const url = `/api/server/files/content?path=${encodeURIComponent(filePath)}&offset=${offset}&limit=${limit}`;
    const resp = await fetch(url);
    if (!resp.ok) throw new Error('Failed to fetch file content');
    return resp.json();
}

export async function fetchFileContent(filePath: string): Promise<string> {
    const resp = await fetch(`/api/server/files/content?path=${encodeURIComponent(filePath)}`);
    if (!resp.ok) throw new Error('Failed to fetch file content');
    const data = await resp.json();
    return data.content;
}

export async function saveFileContent(filePath: string, content: string): Promise<void> {
    const resp = await fetch('/api/server/files/content', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: filePath, content }),
    });
    if (!resp.ok) {
        const err = await resp.json().catch(() => ({ error: 'Failed to save file' }));
        throw new Error(err.error || 'Failed to save file');
    }
}

export interface FileEntry {
    name: string;
    path: string;
    is_dir: boolean;
    size?: number;
    modified_time?: string;
}

export async function fetchServerFiles(basePath: string, path?: string): Promise<FileEntry[]> {
    let url = `/api/server/files?base_path=${encodeURIComponent(basePath)}`;
    if (path) url += `&path=${encodeURIComponent(path)}`;
    const resp = await fetch(url);
    if (!resp.ok) throw new Error('Failed to fetch server files');
    return resp.json();
}

export async function fetchHomeDir(): Promise<string> {
    const resp = await fetch('/api/files/home');
    if (!resp.ok) throw new Error('Failed to fetch home directory');
    const data = await resp.json();
    return data.home_dir;
}
