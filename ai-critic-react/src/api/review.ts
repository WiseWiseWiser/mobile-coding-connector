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

// Unstage a file using git reset HEAD
export async function unstageFile(path: string, dir?: string): Promise<void> {
    const response = await fetch('/api/review/unstage', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path, dir }),
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to unstage file');
    }
}

// Git status file entry
export interface GitStatusFile {
    path: string;
    status: string;
    isStaged: boolean;
}

// Git status result
export interface GitStatusResult {
    branch: string;
    files: GitStatusFile[];
}

// Get git status with staged/unstaged separation
export async function getGitStatus(dir?: string): Promise<GitStatusResult> {
    const response = await fetch('/api/review/status', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dir }),
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to get git status');
    }
    return response.json();
}

// Git commit result
export interface GitCommitResult {
    status: string;
    output: string;
}

// Commit staged changes
export async function gitCommit(message: string, dir?: string): Promise<GitCommitResult> {
    const response = await fetch('/api/review/commit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message, dir }),
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to commit');
    }
    return response.json();
}

// Git branch entry
export interface GitBranch {
    name: string;
    isCurrent: boolean;
    date: string;
}

// List branches sorted by recent commit date
export async function getGitBranches(dir?: string): Promise<GitBranch[]> {
    const response = await fetch('/api/review/branches', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dir }),
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to list branches');
    }
    return response.json();
}

// Push to remote
export async function gitPush(dir?: string): Promise<GitCommitResult> {
    const response = await fetch('/api/review/push', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dir }),
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to push');
    }
    return response.json();
}
