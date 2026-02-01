import type { DiffFile } from './types';

// Parse diff to get original and modified content
export function parseDiffContent(diff: string): { original: string; modified: string } {
    const lines = diff.split('\n');
    const originalLines: string[] = [];
    const modifiedLines: string[] = [];
    let inHunk = false;

    for (const line of lines) {
        if (line.startsWith('@@')) {
            inHunk = true;
            continue;
        }
        if (!inHunk) continue;

        if (line.startsWith('-') && !line.startsWith('---')) {
            originalLines.push(line.substring(1));
        } else if (line.startsWith('+') && !line.startsWith('+++')) {
            modifiedLines.push(line.substring(1));
        } else if (!line.startsWith('\\')) {
            // Context line (no prefix)
            originalLines.push(line.startsWith(' ') ? line.substring(1) : line);
            modifiedLines.push(line.startsWith(' ') ? line.substring(1) : line);
        }
    }

    return {
        original: originalLines.join('\n'),
        modified: modifiedLines.join('\n'),
    };
}

export function getFileLanguage(path: string): string {
    const ext = path.split('.').pop()?.toLowerCase() || '';
    const langMap: Record<string, string> = {
        'ts': 'typescript',
        'tsx': 'typescript',
        'js': 'javascript',
        'jsx': 'javascript',
        'go': 'go',
        'py': 'python',
        'rb': 'ruby',
        'rs': 'rust',
        'java': 'java',
        'c': 'c',
        'cpp': 'cpp',
        'h': 'c',
        'hpp': 'cpp',
        'css': 'css',
        'scss': 'scss',
        'html': 'html',
        'json': 'json',
        'yaml': 'yaml',
        'yml': 'yaml',
        'md': 'markdown',
        'sql': 'sql',
        'sh': 'shell',
        'bash': 'shell',
    };
    return langMap[ext] || 'plaintext';
}

// Group files by directory for tree view
export function groupFilesByDirectory(files: DiffFile[]): Map<string, DiffFile[]> {
    const groups = new Map<string, DiffFile[]>();
    for (const file of files) {
        const dir = file.path.includes('/') ? file.path.substring(0, file.path.lastIndexOf('/')) : '';
        if (!groups.has(dir)) {
            groups.set(dir, []);
        }
        groups.get(dir)!.push(file);
    }
    return groups;
}

export function getFileIcon(path: string): { color: string; letter: string } {
    const ext = path.split('.').pop()?.toLowerCase() || '';
    const iconMap: Record<string, { color: string; letter: string }> = {
        'ts': { color: '#3178c6', letter: 'TS' },
        'tsx': { color: '#3178c6', letter: 'TSX' },
        'js': { color: '#f7df1e', letter: 'JS' },
        'jsx': { color: '#61dafb', letter: 'JSX' },
        'go': { color: '#00add8', letter: 'GO' },
        'py': { color: '#3776ab', letter: 'PY' },
        'rs': { color: '#dea584', letter: 'RS' },
        'java': { color: '#b07219', letter: 'J' },
        'css': { color: '#563d7c', letter: 'CSS' },
        'scss': { color: '#c6538c', letter: 'SCSS' },
        'html': { color: '#e34c26', letter: 'HTML' },
        'json': { color: '#cbcb41', letter: '{ }' },
        'yaml': { color: '#cb171e', letter: 'YML' },
        'yml': { color: '#cb171e', letter: 'YML' },
        'md': { color: '#083fa1', letter: 'MD' },
        'sql': { color: '#e38c00', letter: 'SQL' },
        'sh': { color: '#89e051', letter: 'SH' },
    };
    return iconMap[ext] || { color: '#6b7280', letter: ext.toUpperCase().slice(0, 3) || 'F' };
}

export function getStatusBadge(status: string): { bg: string; color: string; letter: string } {
    switch (status) {
        case 'added': return { bg: '#22c55e', color: '#fff', letter: 'A' };
        case 'deleted': return { bg: '#ef4444', color: '#fff', letter: 'D' };
        case 'renamed': return { bg: '#8b5cf6', color: '#fff', letter: 'R' };
        default: return { bg: '#f59e0b', color: '#fff', letter: 'M' };
    }
}

// Simple markdown formatter for headers and lists
export function formatMarkdown(text: string): string {
    return text
        .replace(/^## (.+)$/gm, '<h2 style="margin-top: 20px; margin-bottom: 10px; font-size: 16px; color: #333; font-weight: 600;">$1</h2>')
        .replace(/^### (.+)$/gm, '<h3 style="margin-top: 15px; margin-bottom: 8px; font-size: 14px; color: #444; font-weight: 600;">$1</h3>')
        .replace(/^\- \[(.+?)\] (.+)$/gm, '<div style="margin-left: 20px; margin-bottom: 5px;"><span style="font-weight: bold; color: #dc2626;">[$1]</span> $2</div>')
        .replace(/^\- (.+)$/gm, '<div style="margin-left: 20px; margin-bottom: 5px;">• $1</div>')
        .replace(/`([^`]+)`/g, '<code style="background: #f3f4f6; padding: 2px 6px; border-radius: 3px; font-family: monospace; font-size: 12px;">$1</code>')
        .replace(/\n\n/g, '<br/><br/>');
}
