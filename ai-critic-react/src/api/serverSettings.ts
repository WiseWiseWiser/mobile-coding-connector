// Server settings API client

export interface ServerConfig {
    project_dir: string;
    auto_detected_dir: string;
    using_explicit_dir: boolean;
}

export async function getServerConfig(): Promise<ServerConfig> {
    const response = await fetch('/api/server/config');
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to get server config');
    }
    return response.json();
}

export async function setServerConfig(projectDir: string): Promise<void> {
    const response = await fetch('/api/server/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project_dir: projectDir }),
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to set server config');
    }
}
