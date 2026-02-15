export function statusBadge(status: string) {
    const cls = `mcc-file-status mcc-file-status-${status}`;
    const label = status === 'added' ? 'A' : status === 'deleted' ? 'D' : 'M';
    return <span className={cls}>{label}</span>;
}

export function formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    const size = bytes / Math.pow(1024, i);
    return `${size.toFixed(1)} ${units[i]}`;
}

export function getFileIcon(filePath: string): string {
    const ext = filePath.split('.').pop()?.toLowerCase() || '';
    const icons: Record<string, string> = {
        ts: '📘',
        tsx: '📘',
        js: '📒',
        jsx: '📒',
        go: '🐹',
        py: '🐍',
        rs: '🦀',
        java: '☕',
        c: '🔧',
        cpp: '🔧',
        h: '🔧',
        css: '🎨',
        scss: '🎨',
        less: '🎨',
        html: '🌐',
        json: '📋',
        yaml: '📋',
        yml: '📋',
        md: '📝',
        txt: '📄',
        png: '🖼️',
        jpg: '🖼️',
        jpeg: '🖼️',
        gif: '🖼️',
        svg: '🖼️',
        pdf: '📕',
        zip: '📦',
        tar: '📦',
        gz: '📦',
    };
    return icons[ext] || '📄';
}

export function getFileSuffix(filePath: string): string {
    const parts = filePath.split('.');
    if (parts.length > 1) {
        return '.' + parts.pop();
    }
    return '';
}
