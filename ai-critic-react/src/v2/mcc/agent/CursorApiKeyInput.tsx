import { useState } from 'react';
import { saveCursorAPIKey, hasCursorAPIKey } from './cursorStorage';

export interface CursorApiKeyInputProps {
    onSuccess?: (message: string) => void;
    onError?: (message: string) => void;
}

export function CursorApiKeyInput({ onSuccess, onError }: CursorApiKeyInputProps) {
    const [apiKey, setApiKey] = useState('');
    const [hasApiKeyStored, setHasApiKeyStored] = useState(() => hasCursorAPIKey());

    const handleSaveApiKey = () => {
        try {
            saveCursorAPIKey(apiKey);
            setHasApiKeyStored(!!apiKey);
            setApiKey(''); // Clear the input after saving
            onSuccess?.(apiKey ? 'API key saved to browser storage' : 'API key cleared');
        } catch (err) {
            onError?.(err instanceof Error ? err.message : 'Failed to save API key');
        }
    };

    return (
        <div className="mcc-agent-settings-field">
            <label className="mcc-agent-settings-label">
                Cursor API Key
            </label>
            <div className="mcc-agent-settings-hint">
                {hasApiKeyStored ? (
                    <span style={{ color: '#86efac' }}>✓ API key is stored in browser</span>
                ) : (
                    <span>Set a custom API key for Cursor agent (stored locally in browser).</span>
                )}
            </div>
            <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                <input
                    type="password"
                    value={apiKey}
                    onChange={e => setApiKey(e.target.value)}
                    onKeyDown={e => e.stopPropagation()}
                    placeholder={hasApiKeyStored ? '••••••••' : 'Enter API key'}
                    autoComplete="new-password"
                    style={{
                        flex: 1,
                        padding: '10px 12px',
                        background: '#1e293b',
                        border: '1px solid #334155',
                        borderRadius: 8,
                        color: '#e2e8f0',
                        fontSize: '14px',
                    }}
                />
                <button
                    onClick={handleSaveApiKey}
                    style={{
                        padding: '10px 16px',
                        background: '#3b82f6',
                        color: '#fff',
                        border: 'none',
                        borderRadius: 8,
                        fontSize: '14px',
                        fontWeight: 600,
                        cursor: 'pointer',
                        whiteSpace: 'nowrap',
                    }}
                >
                    {apiKey ? 'Save' : (hasApiKeyStored ? 'Clear' : 'Save')}
                </button>
            </div>
            <div className="mcc-agent-settings-hint" style={{ marginTop: 8, fontSize: '12px', color: '#64748b' }}>
                Note: The API key will be passed to the agent when starting a new session.
            </div>
        </div>
    );
}
