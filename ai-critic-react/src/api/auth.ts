export const AuthCheckStatuses = {
    Authenticated: 'authenticated',
    Unauthenticated: 'unauthenticated',
    NotInitialized: 'not_initialized',
} as const;

export type AuthCheckStatus = typeof AuthCheckStatuses[keyof typeof AuthCheckStatuses];

export async function checkAuth(): Promise<AuthCheckStatus> {
    const resp = await fetch('/api/auth/check');
    if (resp.ok) {
        return AuthCheckStatuses.Authenticated;
    }
    if (resp.status === 401) {
        const data = await resp.json().catch(() => ({}));
        if (data.error === 'not_initialized') {
            return AuthCheckStatuses.NotInitialized;
        }
        return AuthCheckStatuses.Unauthenticated;
    }
    return AuthCheckStatuses.Authenticated;
}

export async function setupCredential(credential: string): Promise<Response> {
    return fetch('/api/auth/setup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ credential }),
    });
}

export async function login(username: string, password: string): Promise<Response> {
    return fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
    });
}

export interface MaskedCredential {
    masked: string;
}

export async function generateCredential(): Promise<string> {
    const resp = await fetch('/api/auth/credentials/generate', { method: 'POST' });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to generate credential');
    }
    const data = await resp.json();
    return data.credential;
}

export async function addCredentialToken(token: string): Promise<void> {
    const resp = await fetch('/api/auth/credentials/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token }),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to add credential');
    }
}

export async function fetchCredentials(): Promise<MaskedCredential[]> {
    const resp = await fetch('/api/auth/credentials');
    if (!resp.ok) {
        throw new Error('Failed to fetch credentials');
    }
    const data = await resp.json();
    return data.credentials || [];
}

/** Start SSH key test, returns raw Response for SSE streaming. */
export async function testSshKey(host: string, encryptedPrivateKey: string): Promise<Response> {
    const resp = await fetch('/api/ssh-keys/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ host, private_key: encryptedPrivateKey }),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.output || data.error || 'SSH test failed');
    }
    return resp;
}

export interface OAuthConfigStatus {
    configured: boolean;
    client_id?: string;
}

export async function fetchOAuthConfig(): Promise<OAuthConfigStatus> {
    const resp = await fetch('/api/github/oauth-config');
    return resp.json();
}

export async function setOAuthConfig(clientId: string, clientSecret: string): Promise<Response> {
    return fetch('/api/github/oauth-config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ client_id: clientId, client_secret: clientSecret }),
    });
}

export async function exchangeOAuthToken(code: string): Promise<{ access_token?: string; error?: string }> {
    const resp = await fetch('/api/github/oauth-token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code }),
    });
    return resp.json();
}

export interface GithubRepo {
    full_name: string;
    clone_url: string;
    ssh_url: string;
    html_url: string;
    description: string;
    private: boolean;
    language: string;
    updated_at: string;
}

export async function fetchGithubRepos(token: string): Promise<GithubRepo[]> {
    const resp = await fetch('/api/github/repos', {
        headers: { Authorization: `token ${token}` },
    });
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function cloneRepo(body: Record<string, unknown>): Promise<Response> {
    return fetch('/api/github/clone', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
}

export async function fetchPublicKey(): Promise<string> {
    const resp = await fetch('/api/encrypt/public-key');
    if (!resp.ok) {
        throw new Error('Failed to fetch public key');
    }
    return resp.text();
}
