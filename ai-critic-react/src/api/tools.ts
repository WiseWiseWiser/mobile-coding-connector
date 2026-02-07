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
