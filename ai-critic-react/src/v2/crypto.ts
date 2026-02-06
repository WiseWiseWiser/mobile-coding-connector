// RSA-OAEP encryption utility using Web Crypto API

let cachedPublicKey: CryptoKey | null = null;
let cachedPublicKeyPEM: string | null = null;

// EncryptionNotAvailableError is thrown when the server has no encryption keys configured
export class EncryptionNotAvailableError extends Error {
    constructor() {
        super('Server encryption keys not available. Run: go run ./script/crypto/gen');
        this.name = 'EncryptionNotAvailableError';
    }
}

/**
 * Fetches the server's RSA public key (PEM format) and imports it as a CryptoKey.
 * Throws EncryptionNotAvailableError if the server has no keys configured.
 */
async function getPublicKey(): Promise<CryptoKey> {
    if (cachedPublicKey) return cachedPublicKey;

    const resp = await fetch('/api/encrypt/public-key');
    if (!resp.ok) {
        throw new Error('Failed to fetch encryption public key');
    }
    const data = await resp.json();
    const pem: string = data.public_key;

    if (!pem) {
        throw new EncryptionNotAvailableError();
    }

    cachedPublicKeyPEM = pem;

    // Parse PEM to binary
    const pemBody = pem
        .replace(/-----BEGIN PUBLIC KEY-----/, '')
        .replace(/-----END PUBLIC KEY-----/, '')
        .replace(/\s/g, '');
    const binaryStr = atob(pemBody);
    const bytes = new Uint8Array(binaryStr.length);
    for (let i = 0; i < binaryStr.length; i++) {
        bytes[i] = binaryStr.charCodeAt(i);
    }

    // Import as RSA-OAEP key
    const key = await crypto.subtle.importKey(
        'spki',
        bytes.buffer,
        { name: 'RSA-OAEP', hash: 'SHA-256' },
        false,
        ['encrypt']
    );

    cachedPublicKey = key;
    return key;
}

/**
 * Encrypts a string using the server's RSA public key (RSA-OAEP with SHA-256).
 * Since RSA can only encrypt data smaller than the key size, the data is split
 * into chunks. Each encrypted chunk is base64-encoded, and chunks are joined by ".".
 */
export async function encryptWithServerKey(plaintext: string): Promise<string> {
    const key = await getPublicKey();

    // RSA-OAEP with SHA-256 and 3072-bit key: max plaintext = 384 - 66 = 318 bytes
    // Use a conservative chunk size of 190 bytes to be safe with various key sizes
    const MAX_CHUNK_SIZE = 190;
    const encoder = new TextEncoder();
    const data = encoder.encode(plaintext);

    const chunks: string[] = [];
    for (let i = 0; i < data.length; i += MAX_CHUNK_SIZE) {
        const chunk = data.slice(i, i + MAX_CHUNK_SIZE);
        const encrypted = await crypto.subtle.encrypt(
            { name: 'RSA-OAEP' },
            key,
            chunk
        );
        // Convert to base64
        const base64 = btoa(String.fromCharCode(...new Uint8Array(encrypted)));
        chunks.push(base64);
    }

    return chunks.join('.');
}

/**
 * Checks if the encryption public key is available from the server.
 */
export async function isEncryptionAvailable(): Promise<boolean> {
    try {
        await getPublicKey();
        return true;
    } catch {
        return false;
    }
}

/**
 * Returns the cached PEM public key string, or null if not yet loaded.
 */
export function getCachedPublicKeyPEM(): string | null {
    return cachedPublicKeyPEM;
}
