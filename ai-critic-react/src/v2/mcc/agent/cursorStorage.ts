// Local storage helpers for cursor-agent API key
// The API key is stored client-side only for security

const CURSOR_API_KEY_KEY = 'cursor-agent-api-key';

export function loadCursorAPIKey(): string {
    try {
        return localStorage.getItem(CURSOR_API_KEY_KEY) || '';
    } catch {
        return '';
    }
}

export function saveCursorAPIKey(apiKey: string): void {
    try {
        if (apiKey) {
            localStorage.setItem(CURSOR_API_KEY_KEY, apiKey);
        } else {
            localStorage.removeItem(CURSOR_API_KEY_KEY);
        }
    } catch {
        // Ignore storage errors
    }
}

export function hasCursorAPIKey(): boolean {
    return !!loadCursorAPIKey();
}
