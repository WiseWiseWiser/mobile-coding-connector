// Tools diagnostics API client

export interface ToolInfo {
    name: string;
    description: string;
    purpose: string;
    installed: boolean;
    path?: string;
    version?: string;
    install_macos: string;
    install_linux: string;
    install_windows: string;
    auto_install_cmd?: string;
    settings_path?: string;
}

export interface ToolsResponse {
    os: string;
    tools: ToolInfo[];
}

export async function fetchTools(): Promise<ToolsResponse> {
    const resp = await fetch('/api/tools');
    if (!resp.ok) {
        throw new Error('Failed to fetch tools');
    }
    return resp.json();
}

/** Start installing a tool, returns the raw Response for SSE streaming. */
export async function installTool(name: string): Promise<Response> {
    const resp = await fetch(`/api/tools/install?name=${encodeURIComponent(name)}`, {
        method: 'POST',
    });
    if (!resp.ok) {
        const text = await resp.text();
        throw new Error(text || 'Failed to install tool');
    }
    return resp;
}
