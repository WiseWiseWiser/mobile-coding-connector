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

// Git user configuration
export interface GitUserConfig {
    name: string;
    email: string;
}

const GIT_USER_CONFIG_KEY = 'ai-critic-git-user-config';

export function loadGitUserConfig(): GitUserConfig {
    try {
        const data = localStorage.getItem(GIT_USER_CONFIG_KEY);
        if (!data) return { name: '', email: '' };
        return JSON.parse(data) as GitUserConfig;
    } catch {
        return { name: '', email: '' };
    }
}

export function saveGitUserConfig(config: GitUserConfig) {
    localStorage.setItem(GIT_USER_CONFIG_KEY, JSON.stringify(config));
}
