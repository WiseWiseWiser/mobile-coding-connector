import { useState } from 'react';
import type { OpencodeAuthKeyEntry } from '../../../../../api/agents';

export interface ProviderKeysSectionProps {
    authKeys: OpencodeAuthKeyEntry[];
    onSaveKey: (provider: string, key: string) => Promise<void>;
    onDeleteKey: (provider: string) => Promise<void>;
}

export function ProviderKeysSection({ authKeys, onSaveKey, onDeleteKey }: ProviderKeysSectionProps) {
    const [editingKeyProvider, setEditingKeyProvider] = useState<string | null>(null);
    const [editingKeyValue, setEditingKeyValue] = useState('');
    const [newProvider, setNewProvider] = useState('');
    const [newKey, setNewKey] = useState('');
    const [savingKey, setSavingKey] = useState(false);
    const [error, setError] = useState('');

    const handleSaveKey = async (provider: string, key: string) => {
        setSavingKey(true);
        setError('');
        try {
            await onSaveKey(provider, key);
            setEditingKeyProvider(null);
            setEditingKeyValue('');
            setNewProvider('');
            setNewKey('');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save key');
        } finally {
            setSavingKey(false);
        }
    };

    const handleDeleteKey = async (provider: string) => {
        setSavingKey(true);
        setError('');
        try {
            await onDeleteKey(provider);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to delete key');
        } finally {
            setSavingKey(false);
        }
    };

    return (
        <div className="mcc-agent-settings-field" style={{ marginBottom: 20, paddingBottom: 20, borderBottom: '1px solid #334155' }}>
            <label className="mcc-agent-settings-label">
                Provider API Keys
            </label>
            <div className="mcc-agent-settings-hint" style={{ marginBottom: 12, fontSize: '13px', color: '#94a3b8' }}>
                Configure API keys for LLM providers (stored in ~/.local/share/opencode/auth.json).
            </div>

            {error && (
                <div style={{ marginBottom: 8, padding: '6px 10px', background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 4, color: '#fca5a5', fontSize: '12px' }}>
                    {error}
                </div>
            )}

            {authKeys.map(entry => (
                <div key={entry.provider} style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    marginBottom: 8,
                    padding: '8px 12px',
                    background: '#1e293b',
                    border: '1px solid #334155',
                    borderRadius: 8,
                }}>
                    {editingKeyProvider === entry.provider ? (
                        <>
                            <strong style={{ color: '#e2e8f0', minWidth: 100 }}>{entry.provider}</strong>
                            <input
                                type="password"
                                value={editingKeyValue}
                                onChange={e => setEditingKeyValue(e.target.value)}
                                placeholder="Enter new key..."
                                disabled={savingKey}
                                style={{
                                    flex: 1,
                                    padding: '6px 8px',
                                    background: '#0f172a',
                                    border: '1px solid #475569',
                                    borderRadius: 4,
                                    color: '#e2e8f0',
                                    fontSize: '13px',
                                }}
                            />
                            <button
                                onClick={() => handleSaveKey(entry.provider, editingKeyValue)}
                                disabled={savingKey || !editingKeyValue.trim()}
                                style={{ padding: '4px 10px', fontSize: '12px', background: '#22c55e', border: 'none', borderRadius: 4, color: '#fff', cursor: 'pointer' }}
                            >
                                Save
                            </button>
                            <button
                                onClick={() => { setEditingKeyProvider(null); setEditingKeyValue(''); }}
                                disabled={savingKey}
                                style={{ padding: '4px 10px', fontSize: '12px', background: 'transparent', border: '1px solid #475569', borderRadius: 4, color: '#94a3b8', cursor: 'pointer' }}
                            >
                                Cancel
                            </button>
                        </>
                    ) : (
                        <>
                            <strong style={{ color: '#e2e8f0', minWidth: 100 }}>{entry.provider}</strong>
                            <span style={{ flex: 1, fontFamily: 'monospace', fontSize: '12px', color: '#94a3b8' }}>
                                {entry.masked_key || '(empty)'}
                            </span>
                            <button
                                onClick={() => { setEditingKeyProvider(entry.provider); setEditingKeyValue(''); }}
                                style={{ padding: '4px 10px', fontSize: '12px', background: 'transparent', border: '1px solid #475569', borderRadius: 4, color: '#60a5fa', cursor: 'pointer' }}
                            >
                                Edit
                            </button>
                            <button
                                onClick={() => handleDeleteKey(entry.provider)}
                                disabled={savingKey}
                                style={{ padding: '4px 10px', fontSize: '12px', background: 'transparent', border: '1px solid #475569', borderRadius: 4, color: '#f87171', cursor: 'pointer' }}
                            >
                                Delete
                            </button>
                        </>
                    )}
                </div>
            ))}

            {/* Add new provider */}
            <div style={{
                display: 'flex',
                gap: 8,
                marginTop: authKeys.length > 0 ? 12 : 0,
            }}>
                <input
                    type="text"
                    value={newProvider}
                    onChange={e => setNewProvider(e.target.value)}
                    placeholder="Provider name (e.g. openrouter)"
                    disabled={savingKey}
                    style={{
                        width: 180,
                        padding: '8px 10px',
                        background: '#1e293b',
                        border: '1px solid #334155',
                        borderRadius: 6,
                        color: '#e2e8f0',
                        fontSize: '13px',
                    }}
                />
                <input
                    type="password"
                    value={newKey}
                    onChange={e => setNewKey(e.target.value)}
                    placeholder="API key"
                    disabled={savingKey}
                    style={{
                        flex: 1,
                        padding: '8px 10px',
                        background: '#1e293b',
                        border: '1px solid #334155',
                        borderRadius: 6,
                        color: '#e2e8f0',
                        fontSize: '13px',
                    }}
                />
                <button
                    onClick={() => handleSaveKey(newProvider.trim(), newKey)}
                    disabled={savingKey || !newProvider.trim() || !newKey.trim()}
                    style={{
                        padding: '8px 16px',
                        fontSize: '13px',
                        background: newProvider.trim() && newKey.trim() ? '#3b82f6' : '#475569',
                        border: 'none',
                        borderRadius: 6,
                        color: '#fff',
                        fontWeight: 500,
                        cursor: (savingKey || !newProvider.trim() || !newKey.trim()) ? 'not-allowed' : 'pointer',
                        opacity: (savingKey || !newProvider.trim() || !newKey.trim()) ? 0.6 : 1,
                    }}
                >
                    Add
                </button>
            </div>
        </div>
    );
}
