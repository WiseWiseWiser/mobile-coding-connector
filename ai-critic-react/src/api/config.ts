export interface ServerConfig {
    enableMockupInMenu: boolean;
}

export async function fetchServerConfig(): Promise<ServerConfig> {
    const response = await fetch('/api/config');
    if (!response.ok) {
        throw new Error(`Failed to fetch server config: ${response.statusText}`);
    }
    return response.json();
}
