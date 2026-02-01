import type { 
    GitDiffResult, 
    ConfigResponse,
} from '../components/code-review/types';

// Get configuration including initial directory and available providers/models
export async function getConfig(): Promise<ConfigResponse> {
    const response = await fetch('/api/review/config');
    return response.json();
}

// Get git diff for a directory
export async function getDiff(dir?: string): Promise<GitDiffResult> {
    const response = await fetch('/api/review/diff', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dir }),
    });
    return response.json();
}

// Stage a file using git add
export async function stageFile(path: string, dir?: string): Promise<void> {
    const response = await fetch('/api/review/stage', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path, dir }),
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to stage file');
    }
}
