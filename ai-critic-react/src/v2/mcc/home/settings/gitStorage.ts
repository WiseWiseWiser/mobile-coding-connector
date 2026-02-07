// Shared localStorage helpers for SSH keys and GitHub token.

export interface SSHKey {
    id: string;
    name: string;
    host: string;
    privateKey: string;
    createdAt: string;
}

const SSH_KEYS_STORAGE_KEY = 'ai-critic-ssh-keys';

export function loadSSHKeys(): SSHKey[] {
    try {
        const data = localStorage.getItem(SSH_KEYS_STORAGE_KEY);
        if (!data) return [];
        return JSON.parse(data) as SSHKey[];
    } catch {
        return [];
    }
}

export function saveSSHKeys(keys: SSHKey[]) {
    localStorage.setItem(SSH_KEYS_STORAGE_KEY, JSON.stringify(keys));
}

const GITHUB_TOKEN_STORAGE_KEY = 'ai-critic-github-token';

export function loadGitHubToken(): string {
    return localStorage.getItem(GITHUB_TOKEN_STORAGE_KEY) || '';
}

export function saveGitHubToken(token: string) {
    if (token) {
        localStorage.setItem(GITHUB_TOKEN_STORAGE_KEY, token);
    } else {
        localStorage.removeItem(GITHUB_TOKEN_STORAGE_KEY);
    }
}
