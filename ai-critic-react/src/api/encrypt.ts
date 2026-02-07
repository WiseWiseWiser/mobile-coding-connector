// Encryption key pair API client

export interface EncryptKeyStatus {
    exists: boolean;
    valid: boolean;
    error?: string;
    private_key_path: string;
    public_key_path: string;
}

export async function fetchEncryptKeyStatus(): Promise<EncryptKeyStatus> {
    const resp = await fetch('/api/encrypt/status');
    if (!resp.ok) {
        throw new Error('Failed to fetch encryption key status');
    }
    return resp.json();
}

export async function generateEncryptKeys(): Promise<EncryptKeyStatus> {
    const resp = await fetch('/api/encrypt/generate', { method: 'POST' });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to generate encryption keys');
    }
    return resp.json();
}
