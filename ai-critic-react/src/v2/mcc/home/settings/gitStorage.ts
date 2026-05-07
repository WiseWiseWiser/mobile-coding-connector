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

// Git user configurations
export interface LegacyGitUserConfig {
    name: string;
    email: string;
}

export interface GitUserConfig {
    id: string;
    name: string;
    email: string;
    createdAt: string;
}

const GIT_USER_CONFIG_KEY = 'ai-critic-git-user-config';
const GIT_USER_CONFIGS_KEY = 'ai-critic-git-user-configs';

function stableGitUserConfigId(name: string, email: string): string {
    let hash = 5381;
    const input = `${name}\n${email}`;
    for (let i = 0; i < input.length; i++) {
        hash = ((hash << 5) + hash) ^ input.charCodeAt(i);
    }
    return `git-user-${(hash >>> 0).toString(36)}`;
}

export function createGitUserConfig(name: string, email: string): GitUserConfig {
    return {
        id: Date.now().toString(36) + Math.random().toString(36).slice(2, 8),
        name: name.trim(),
        email: email.trim(),
        createdAt: new Date().toISOString(),
    };
}

function normalizeGitUserConfig(config: Partial<GitUserConfig> | LegacyGitUserConfig | null | undefined): GitUserConfig | null {
    if (!config) return null;
    const name = (config.name || '').trim();
    const email = (config.email || '').trim();
    if (!name || !email) return null;
    return {
        id: 'id' in config && config.id ? config.id : stableGitUserConfigId(name, email),
        name,
        email,
        createdAt: 'createdAt' in config && config.createdAt ? config.createdAt : new Date().toISOString(),
    };
}

export function normalizeGitUserConfigs(configs: Array<Partial<GitUserConfig> | LegacyGitUserConfig>): GitUserConfig[] {
    const seen = new Set<string>();
    const normalized: GitUserConfig[] = [];
    for (const config of configs) {
        const item = normalizeGitUserConfig(config);
        if (!item || seen.has(item.id)) continue;
        seen.add(item.id);
        normalized.push(item);
    }
    return normalized;
}

export function loadGitUserConfigs(): GitUserConfig[] {
    try {
        const data = localStorage.getItem(GIT_USER_CONFIGS_KEY);
        if (data !== null) {
            const parsed = JSON.parse(data);
            return Array.isArray(parsed) ? normalizeGitUserConfigs(parsed) : [];
        }

        const legacyData = localStorage.getItem(GIT_USER_CONFIG_KEY);
        if (!legacyData) return [];
        const legacyConfig = normalizeGitUserConfig(JSON.parse(legacyData));
        if (!legacyConfig) return [];
        saveGitUserConfigs([legacyConfig]);
        return [legacyConfig];
    } catch {
        return [];
    }
}

export function saveGitUserConfigs(configs: Array<Partial<GitUserConfig> | LegacyGitUserConfig>) {
    const normalized = normalizeGitUserConfigs(configs);
    localStorage.setItem(GIT_USER_CONFIGS_KEY, JSON.stringify(normalized));

    // Keep the old single-config key readable for older exports and app versions.
    if (normalized.length > 0) {
        const first = normalized[0];
        localStorage.setItem(GIT_USER_CONFIG_KEY, JSON.stringify({ name: first.name, email: first.email }));
    } else {
        localStorage.removeItem(GIT_USER_CONFIG_KEY);
    }
}

async function readGitUserConfigAPIError(resp: Response): Promise<string> {
    const data = await resp.json().catch(() => null);
    return data?.error || `Request failed: ${resp.status}`;
}

export async function loadGitUserConfigsFromServer(): Promise<GitUserConfig[]> {
    const resp = await fetch('/api/settings/git-user-configs');
    if (!resp.ok) {
        throw new Error(await readGitUserConfigAPIError(resp));
    }
    const data = await resp.json();
    const configs = Array.isArray(data) ? normalizeGitUserConfigs(data) : [];
    saveGitUserConfigs(configs);
    return configs;
}

export async function saveGitUserConfigsToServer(configs: Array<Partial<GitUserConfig> | LegacyGitUserConfig>): Promise<GitUserConfig[]> {
    const normalized = normalizeGitUserConfigs(configs);
    const resp = await fetch('/api/settings/git-user-configs', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ configs: normalized }),
    });
    if (!resp.ok) {
        throw new Error(await readGitUserConfigAPIError(resp));
    }
    const data = await resp.json();
    const saved = Array.isArray(data) ? normalizeGitUserConfigs(data) : normalized;
    saveGitUserConfigs(saved);
    return saved;
}

export function loadGitUserConfig(): LegacyGitUserConfig {
    const first = loadGitUserConfigs()[0];
    return first ? { name: first.name, email: first.email } : { name: '', email: '' };
}

export function saveGitUserConfig(config: Partial<GitUserConfig> | LegacyGitUserConfig) {
    const normalized = normalizeGitUserConfig(config);
    saveGitUserConfigs(normalized ? [normalized] : []);
}

export function formatGitUserConfig(config: Pick<GitUserConfig, 'name' | 'email'>): string {
    return `${config.name} <${config.email}>`;
}
