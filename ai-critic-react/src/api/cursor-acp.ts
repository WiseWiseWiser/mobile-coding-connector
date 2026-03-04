const CURSOR_ACP_PREFIX = '/api/agent/acp/cursor';

// --- Types ---

export interface EffectivePathInfo {
    found: boolean;
    effective_path?: string;
    error?: string;
}

export interface CursorACPSettings {
    api_key: string;
    binary_path: string;
    effective_path?: EffectivePathInfo;
    default_model?: string;
    default_model_name?: string;
}

export interface SaveCursorACPSettingsRequest {
    api_key: string;
    binary_path: string;
    default_model: string;
    default_model_name: string;
}

export interface CursorACPModelInfo {
    id: string;
    name: string;
    providerId: string;
    providerName: string;
    is_current?: boolean;
}

export interface CursorACPStatusInfo {
    available: boolean;
    connected: boolean;
    session_id?: string;
    cwd?: string;
    projectDir?: string;
    model?: string;
    message?: string;
}

export interface CursorACPSessionInfo {
    id: string;
    dir?: string;
    status?: string;
}

export interface CursorACPSessionSettings {
    trustWorkspace: boolean;
    yoloMode: boolean;
}

export interface SaveSessionSettingsRequest {
    sessionId: string;
    trustWorkspace?: boolean;
    yoloMode?: boolean;
}

export interface ChatMessage {
    role: string;
    content: string;
}

export interface ConnectRequest {
    cwd?: string;
    dir?: string;
    sessionId?: string;
    model?: string;
    projectName?: string;
    worktreeId?: string;
    debug?: boolean;
}

export interface PromptRequest {
    sessionId: string;
    prompt: string;
    model?: string;
    debug?: boolean;
}

// --- Generic ACP API (for reuse with different prefixes) ---

export function createACPAPI(prefix: string) {
    return {
        async fetchStatus(): Promise<CursorACPStatusInfo> {
            const resp = await fetch(`${prefix}/status`);
            return resp.json();
        },

        async fetchModels(): Promise<CursorACPModelInfo[]> {
            const resp = await fetch(`${prefix}/models`);
            if (!resp.ok) {
                const errData = await resp.json().catch(() => ({}));
                throw new Error(errData.message || `Failed to load models: ${resp.status}`);
            }
            const data = await resp.json();
            if (Array.isArray(data)) {
                return data;
            }
            if (data.models && Array.isArray(data.models)) {
                return data.models;
            }
            return [];
        },

        async fetchSessions(): Promise<CursorACPSessionInfo[]> {
            const resp = await fetch(`${prefix}/sessions`);
            const data = await resp.json();
            return Array.isArray(data) ? data : [];
        },

        async fetchSession(sessionId: string): Promise<CursorACPSessionInfo> {
            const resp = await fetch(`${prefix}/session?sessionId=${encodeURIComponent(sessionId)}`);
            return resp.json();
        },

        async fetchSessionMessages(sessionId: string): Promise<ChatMessage[]> {
            const resp = await fetch(`${prefix}/session/messages?sessionId=${encodeURIComponent(sessionId)}`);
            if (!resp.ok) return [];
            const data = await resp.json();
            return Array.isArray(data) ? data : [];
        },

        async saveSessionMessages(sessionId: string, messages: ChatMessage[]): Promise<void> {
            await fetch(`${prefix}/session/messages`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ sessionId, messages }),
            });
        },

        async connect(body: ConnectRequest): Promise<Response> {
            return fetch(`${prefix}/connect`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
            });
        },

        async sendPrompt(body: PromptRequest, signal?: AbortSignal): Promise<Response> {
            const resp = await fetch(`${prefix}/prompt`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
                signal,
            });
            if (!resp.ok) {
                const errData = await resp.json().catch(() => ({}));
                throw new Error(errData.message || `Request failed (${resp.status})`);
            }
            return resp;
        },

        async cancel(sessionId: string): Promise<void> {
            await fetch(`${prefix}/cancel`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ sessionId }),
            });
        },

        async disconnect(): Promise<void> {
            await fetch(`${prefix}/disconnect`, { method: 'POST' });
        },

        async updateSessionModel(sessionId: string, model: string): Promise<void> {
            await fetch(`${prefix}/session/model`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ sessionId, model }),
            });
        },
    };
}

// --- Cursor-specific API ---

const cursorAPI = createACPAPI(CURSOR_ACP_PREFIX);

export async function fetchCursorACPSettings(): Promise<CursorACPSettings> {
    const resp = await fetch(`${CURSOR_ACP_PREFIX}/settings`);
    return resp.json();
}

export async function saveCursorACPSettings(req: SaveCursorACPSettingsRequest): Promise<void> {
    const resp = await fetch(`${CURSOR_ACP_PREFIX}/settings`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || `Failed (${resp.status})`);
    }
}

export async function fetchCursorACPSessionSettings(sessionId: string): Promise<CursorACPSessionSettings> {
    const resp = await fetch(`${CURSOR_ACP_PREFIX}/session/settings?sessionId=${encodeURIComponent(sessionId)}`);
    if (!resp.ok) {
        throw new Error(`Failed to load settings: ${resp.status}`);
    }
    return resp.json();
}

export async function saveCursorACPSessionSettings(req: SaveSessionSettingsRequest): Promise<void> {
    const resp = await fetch(`${CURSOR_ACP_PREFIX}/session/settings`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || `Failed to save: ${resp.status}`);
    }
}

export async function validateCursorAPIKey(apiKey: string): Promise<Response> {
    const resp = await fetch(`${CURSOR_ACP_PREFIX}/settings/validate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ api_key: apiKey }),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || `Validation failed (${resp.status})`);
    }
    return resp;
}

export { cursorAPI };
