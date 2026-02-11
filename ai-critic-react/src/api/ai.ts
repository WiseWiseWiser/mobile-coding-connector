export interface AIProvider {
    name: string;
    base_url: string;
    api_key: string;
}

export interface AIModel {
    provider: string;
    model: string;
    display_name?: string;
    max_tokens?: number;
}

export interface AIConfig {
    providers: AIProvider[];
    models: AIModel[];
    default_provider?: string;
    default_model?: string;
    using_new_file: boolean;
}

export async function fetchAIConfig(): Promise<AIConfig> {
    const res = await fetch('/api/ai-config');
    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Failed to fetch AI config' }));
        throw new Error(err.error || 'Failed to fetch AI config');
    }
    return res.json();
}

export async function saveAIConfig(config: Omit<AIConfig, 'using_new_file'>): Promise<void> {
    const res = await fetch('/api/ai-config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
    });
    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Failed to save AI config' }));
        throw new Error(err.error || 'Failed to save AI config');
    }
}
