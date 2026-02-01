import type { ProviderInfo, ModelInfo } from './types';

interface ModelSelectorProps {
    providers: ProviderInfo[];
    models: ModelInfo[];
    selectedProvider: string;
    selectedModel: string;
    onProviderChange: (provider: string) => void;
    onModelChange: (model: string) => void;
}

export function ModelSelector({
    providers,
    models,
    selectedProvider,
    selectedModel,
    onProviderChange,
    onModelChange,
}: ModelSelectorProps) {
    // Filter models by selected provider
    const filteredModels = selectedProvider 
        ? models.filter(m => m.provider === selectedProvider)
        : models;

    return (
        <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
            <select
                value={selectedProvider}
                onChange={(e) => {
                    onProviderChange(e.target.value);
                    // Reset model when provider changes
                    const providerModels = models.filter(m => m.provider === e.target.value);
                    if (providerModels.length > 0) {
                        onModelChange(providerModels[0].model);
                    }
                }}
                style={{
                    padding: '6px 10px',
                    fontSize: '12px',
                    border: '1px solid #d1d5db',
                    borderRadius: '4px',
                    backgroundColor: '#fff',
                    minWidth: '120px',
                }}
            >
                <option value="">Select Provider</option>
                {providers.map(p => (
                    <option key={p.name} value={p.name}>{p.name}</option>
                ))}
            </select>

            <select
                value={selectedModel}
                onChange={(e) => onModelChange(e.target.value)}
                style={{
                    padding: '6px 10px',
                    fontSize: '12px',
                    border: '1px solid #d1d5db',
                    borderRadius: '4px',
                    backgroundColor: '#fff',
                    minWidth: '150px',
                }}
                disabled={!selectedProvider}
            >
                <option value="">Select Model</option>
                {filteredModels.map(m => (
                    <option key={`${m.provider}-${m.model}`} value={m.model}>
                        {m.displayName || m.model}
                    </option>
                ))}
            </select>
        </div>
    );
}
