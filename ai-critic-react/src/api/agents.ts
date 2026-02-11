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

export interface AgentSessionsResponse {
    sessions: AgentSessionInfo[];
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
}

export interface OpencodeSession {
    id: string;
    created_at?: string;
    firstMessage?: string;
}

export interface OpencodeSessionsResponse {
    items: OpencodeSession[];
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
}

// ---- API functions ----

export async function fetchAgents(): Promise<AgentDef[]> {
    const resp = await fetch('/api/agents');
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

export async function fetchAgentSessions(page?: number, pageSize?: number): Promise<AgentSessionInfo[]> {
    const params = new URLSearchParams();
    if (page) params.set('page', page.toString());
    if (pageSize) params.set('page_size', pageSize.toString());
    
    const url = params.toString() ? `/api/agents/sessions?${params}` : '/api/agents/sessions';
    const resp = await fetch(url);
    const data = await resp.json();
    
    // Handle both paginated and legacy response formats
    if (data.sessions && Array.isArray(data.sessions)) {
        return data.sessions;
    }
    return Array.isArray(data) ? data : [];
}

export async function fetchAgentSessionsPaginated(page: number = 1, pageSize: number = 20): Promise<AgentSessionsResponse> {
    const params = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
    });
    
    const resp = await fetch(`/api/agents/sessions?${params}`);
    return resp.json();
}

export interface LaunchAgentOptions {
    agentId: string;
    projectDir: string;
    apiKey?: string; // Optional API key (e.g., for cursor-agent)
}

export async function launchAgentSession(agentId: string, projectDir: string, apiKey?: string): Promise<AgentSessionInfo> {
    const body: Record<string, string> = { agent_id: agentId, project_dir: projectDir };
    if (apiKey) {
        body.api_key = apiKey;
    }
    const resp = await fetch('/api/agents/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
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

export async function listOpencodeSessions(sessionId: string, page?: number, pageSize?: number): Promise<{ id: string }[]> {
    const params = new URLSearchParams();
    if (page) params.set('page', page.toString());
    if (pageSize) params.set('page_size', pageSize.toString());
    
    const url = params.toString() ? `${agentProxyBase(sessionId)}/session?${params}` : `${agentProxyBase(sessionId)}/session`;
    const resp = await fetch(url);
    const data = await resp.json();
    
    // Handle both paginated and legacy response formats
    if (data.items && Array.isArray(data.items)) {
        return data.items;
    }
    return Array.isArray(data) ? data : [];
}

export async function listOpencodeSessionsPaginated(sessionId: string, page: number = 1, pageSize: number = 50): Promise<OpencodeSessionsResponse> {
    const params = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
    });
    
    const resp = await fetch(`${agentProxyBase(sessionId)}/session?${params}`);
    return resp.json();
}

export async function createOpencodeSession(sessionId: string, model?: { modelID: string; providerID: string }): Promise<{ id: string }> {
    const body: Record<string, unknown> = {};
    if (model) {
        body.model = `${model.providerID}/${model.modelID}`;
    }
    const resp = await fetch(`${agentProxyBase(sessionId)}/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
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

export async function updateAgentConfig(sessionId: string, config: { model: { modelID: string } }): Promise<void> {
    const resp = await fetch(`${agentProxyBase(sessionId)}/config`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to update config');
    }
    const data = await resp.json().catch(() => null);
    if (data && data.success === false) {
        const errMsg = Array.isArray(data.error)
            ? data.error.map((e: { message?: string }) => e.message).join('; ')
            : String(data.error || 'Failed to update config');
        throw new Error(errMsg);
    }
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

export async function sendPromptAsync(
    sessionId: string,
    opencodeSID: string,
    text: string,
    model?: { modelID: string; providerID: string }
): Promise<void> {
    const body: Record<string, unknown> = {
        parts: [{ type: 'text', text }],
    };
    if (model) {
        body.model = model;
    }
    await fetch(`${agentProxyBase(sessionId)}/session/${opencodeSID}/prompt_async`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
}

/** Build the SSE event URL for a given agent session */
export function agentEventUrl(sessionId: string): string {
    return `${agentProxyBase(sessionId)}/event`;
}

// ---- Agent Settings ----

export interface AgentSettings {
    prompt_append_message: string;
    followup_append_message: string;
}

export async function fetchAgentSettings(sessionId: string): Promise<AgentSettings> {
    const resp = await fetch(`${agentProxyBase(sessionId)}/settings`);
    return resp.json();
}

export async function updateAgentSettings(sessionId: string, settings: AgentSettings): Promise<AgentSettings> {
    const resp = await fetch(`${agentProxyBase(sessionId)}/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
    });
    return resp.json();
}

// ---- Agent Templates ----

export interface AgentTemplate {
    id: string;
    name: string;
    content: string;
}

export async function fetchAgentTemplates(sessionId: string): Promise<AgentTemplate[]> {
    const resp = await fetch(`${agentProxyBase(sessionId)}/templates`);
    const data = await resp.json();
    return Array.isArray(data) ? data : [];
}

// ---- Agent Path Configuration ----

export interface AgentPathConfig {
    binary_path?: string;
}

export interface AgentsPathConfig {
    agents: Record<string, AgentPathConfig>;
}

export async function fetchAgentPathConfig(agentId: string): Promise<AgentPathConfig> {
    const resp = await fetch(`/api/agents/config?agent_id=${encodeURIComponent(agentId)}`);
    return resp.json();
}

export async function updateAgentPathConfig(agentId: string, binaryPath: string): Promise<void> {
    const resp = await fetch(`/api/agents/config?agent_id=${encodeURIComponent(agentId)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ binary_path: binaryPath }),
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to update agent config');
    }
}


export interface AgentEffectivePath {
    effective_path: string;
    found: boolean;
    error: string;
}

export async function fetchAgentEffectivePath(agentId: string): Promise<AgentEffectivePath> {
    const resp = await fetch(`/api/agents/effective-path?agent_id=${encodeURIComponent(agentId)}`);
    return resp.json();
}

// ---- OpenCode Settings ----

export interface OpencodeSettings {
    model?: string;
    default_domain?: string;
    web_server?: {
        enabled: boolean;
        port: number;
        exposed_domain?: string;
        password?: string;
    };
}

export async function fetchOpencodeSettings(): Promise<OpencodeSettings> {
    const resp = await fetch('/api/agents/opencode/settings');
    return resp.json();
}

export async function updateOpencodeSettings(settings: OpencodeSettings): Promise<void> {
    const resp = await fetch('/api/agents/opencode/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to update settings');
    }
}

// ---- OpenCode Web Server Status ----

export interface OpencodeWebStatus {
    running: boolean;
    port: number;
    domain: string;
    port_mapped: boolean;
    config_path: string;
}

export async function fetchOpencodeWebStatus(): Promise<OpencodeWebStatus> {
    const resp = await fetch('/api/agents/opencode/web-status');
    return resp.json();
}

// ---- OpenCode Web Server Control ----

export interface WebServerControlRequest {
    action: 'start' | 'stop';
}

export interface WebServerControlResponse {
    success: boolean;
    message: string;
    running: boolean;
}

export async function controlOpencodeWebServer(action: 'start' | 'stop'): Promise<WebServerControlResponse> {
    const resp = await fetch('/api/agents/opencode/web-server/control', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action }),
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to control web server');
    }
    return resp.json();
}

/** Control web server with streaming (returns Response for SSE consumption). */
export function controlOpencodeWebServerStreaming(action: 'start' | 'stop'): Promise<Response> {
    return fetch('/api/agents/opencode/web-server/control', {
        method: 'POST',
        headers: { 
            'Content-Type': 'application/json',
            'Accept': 'text/event-stream',
        },
        body: JSON.stringify({ action }),
    });
}

// ---- OpenCode Web Server Domain Mapping ----

export interface MapDomainRequest {
    provider?: string;
}

export interface MapDomainResponse {
    success: boolean;
    message: string;
    public_url?: string;
}

export async function mapOpencodeDomain(provider?: string): Promise<MapDomainResponse> {
    const resp = await fetch('/api/agents/opencode/web-server/domain-map', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider }),
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to map domain');
    }
    return resp.json();
}

export async function unmapOpencodeDomain(): Promise<MapDomainResponse> {
    const resp = await fetch('/api/agents/opencode/web-server/domain-map', {
        method: 'DELETE',
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to unmap domain');
    }
    return resp.json();
}

// ---- OpenCode Auth Status ----

export interface OpencodeAuthProvider {
    name: string;
    has_api_key: boolean;
}

export interface OpencodeAuthStatus {
    authenticated: boolean;
    providers: OpencodeAuthProvider[];
    config_path: string;
}

export async function fetchOpencodeAuthStatus(): Promise<OpencodeAuthStatus> {
    const resp = await fetch('/api/agents/opencode/auth');
    return resp.json();
}
