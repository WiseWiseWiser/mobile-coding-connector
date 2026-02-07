// Terminal configuration API client

export interface TerminalConfig {
    extra_paths: string[];
    shell?: string;       // shell path or name (default: "bash")
    shell_flags?: string[]; // shell flags (default: ["-i"])
}

export async function fetchTerminalConfig(): Promise<TerminalConfig> {
    const resp = await fetch('/api/terminal/config');
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to load terminal config');
    }
    return resp.json();
}

export async function saveTerminalConfig(config: TerminalConfig): Promise<void> {
    const resp = await fetch('/api/terminal/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
    });
    if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to save terminal config');
    }
}
