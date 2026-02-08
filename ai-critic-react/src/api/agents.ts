// ---- Types ----

export interface AgentDef {
    id: string;
    name: string;
    description: string;
    command: string;
    installed: boolean;
    headless: boolean;
}

export const AgentSessionStatuses = {
    Starting: 'starting',
    Running: 'running',
    Stopped: 'stopped',
    Error: 'error',
} as const;

export type AgentSessionStatus = typeof AgentSessionStatuses[keyof typeof AgentSessionStatuses];

export interface AgentSessionInfo {
    id: string;
    agent_id: string;
    agent_name: string;
    project_dir: string;
    port: number;
    created_at: string;
    status: AgentSessionStatus;
    error?: string;
}

// ---- API functions ----

export async function fetchAgents(): Promise<AgentDef[]> {
    const resp = await fetch('/api/agents');
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function fetchAgentSessions(): Promise<AgentSessionInfo[]> {
    const resp = await fetch('/api/agents/sessions');
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function launchAgentSession(agentId: string, projectDir: string): Promise<AgentSessionInfo> {
    const resp = await fetch('/api/agents/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent_id: agentId, project_dir: projectDir }),
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to launch agent');
    }
    return resp.json();
}

export async function stopAgentSession(sessionId: string): Promise<void> {
    await fetch(`/api/agents/sessions?id=${encodeURIComponent(sessionId)}`, { method: 'DELETE' });
}

/** Build the proxy base URL for a given agent session */
export function agentProxyBase(sessionId: string): string {
    return `/api/agents/sessions/${sessionId}/proxy`;
}

export async function listOpencodeSessions(sessionId: string): Promise<{ id: string }[]> {
    const resp = await fetch(`${agentProxyBase(sessionId)}/session`);
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function createOpencodeSession(sessionId: string): Promise<{ id: string }> {
    const resp = await fetch(`${agentProxyBase(sessionId)}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: '{}',
    });
    return resp.json();
}

/**
 * Get or create an opencode session for the given agent session.
 * Reuses the first existing session if any, otherwise creates a new one.
 */
export async function getOrCreateOpencodeSession(sessionId: string): Promise<{ id: string }> {
    const existing = await listOpencodeSessions(sessionId);
    if (existing.length > 0) return existing[0];
    return createOpencodeSession(sessionId);
}

/** OpenCode config from GET /config */
export interface OpencodeConfig {
    model?: {
        modelID: string;
        providerID: string;
    };
    [key: string]: unknown;
}

export async function fetchOpencodeConfig(sessionId: string): Promise<OpencodeConfig> {
    const resp = await fetch(`${agentProxyBase(sessionId)}/config`);
    return resp.json();
}

/** OpenCode provider model info */
export interface OpencodeModelInfo {
    id: string;
    name: string;
    limit: {
        context: number;
        output: number;
    };
    [key: string]: unknown;
}

/** GET /config/providers - returns providers with models and defaults */
export async function fetchOpencodeProviders(sessionId: string): Promise<{
    providers: Array<{
        id: string;
        name: string;
        models: Record<string, OpencodeModelInfo>;
    }>;
    default: Record<string, string>;
}> {
    const resp = await fetch(`${agentProxyBase(sessionId)}/config/providers`);
    return resp.json();
}

export interface AgentMessage {
    info: {
        id: string;
        role: string;
        time: string | number | { created: number; completed?: number };
        // Fields from AssistantMessage
        modelID?: string;
        providerID?: string;
        cost?: number;
        tokens?: {
            input: number;
            output: number;
            reasoning: number;
            cache: { read: number; write: number };
        };
    };
    parts: MessagePart[];
}

export interface MessagePart {
    id?: string;
    messageID?: string;
    type: string;
    content?: string;
    text?: string;
    tool?: string;
    callID?: string;
    input?: unknown;
    output?: string;
    state?: string | ToolState;
    title?: string;
    thinking?: string;
    reasoning?: string;
}

/** Tool state from OpenCode SDK */
export interface ToolState {
    status: 'pending' | 'running' | 'completed' | 'error';
    input?: unknown;
    output?: string;
    title?: string;
    error?: string;
    time?: {
        start?: number;
        end?: number;
    };
}

export async function fetchMessages(sessionId: string, opencodeSID: string): Promise<AgentMessage[]> {
    const resp = await fetch(`${agentProxyBase(sessionId)}/session/${opencodeSID}/message`);
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function sendPromptAsync(sessionId: string, opencodeSID: string, text: string): Promise<void> {
    await fetch(`${agentProxyBase(sessionId)}/session/${opencodeSID}/prompt_async`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            parts: [{ type: 'text', text }],
        }),
    });
}

/** Build the SSE event URL for a given agent session */
export function agentEventUrl(sessionId: string): string {
    return `${agentProxyBase(sessionId)}/event`;
}
