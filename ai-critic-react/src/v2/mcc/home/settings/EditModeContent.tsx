import { DomainProviders, fetchRandomDomain } from '../../../../api/domains';
import type { DomainEntry } from '../../../../api/domains';
import { FlexInput } from '../../../../pure-view/FlexInput';

export interface EditModeContentProps {
    editTunnelName: string;
    onEditTunnelName: (v: string) => void;
    editEntries: DomainEntry[];
    saving: boolean;
    error: string | null;
    showAddForm: boolean;
    newDomain: string;
    newProvider: string;
    onSetShowAddForm: (v: boolean) => void;
    onSetNewDomain: (v: string) => void;
    onSetNewProvider: (v: string) => void;
    onSetError: (v: string | null) => void;
    onUpdateEntry: (index: number, field: keyof DomainEntry, value: string) => void;
    onAddEntry: () => void;
    onRemoveEntry: (index: number) => void;
    onSave: () => void;
    onCancel: () => void;
}

export function EditModeContent({
    editTunnelName, onEditTunnelName,
    editEntries, saving, error,
    showAddForm, newDomain, newProvider,
    onSetShowAddForm, onSetNewDomain, onSetNewProvider, onSetError,
    onUpdateEntry, onAddEntry, onRemoveEntry, onSave, onCancel,
}: EditModeContentProps) {
    return (
        <div className="diagnose-webaccess-card">
            {editEntries.length > 0 && (
                <div className="diagnose-webaccess-list">
                    {editEntries.map((entry, i) => (
                        <div key={i} className="diagnose-webaccess-row">
                            <div className="diagnose-webaccess-add-row">
                                <label className="diagnose-webaccess-add-label">Domain</label>
                                <FlexInput
                                    inputClassName="diagnose-webaccess-input"
                                    placeholder="e.g. myapp.example.com"
                                    value={entry.domain}
                                    onChange={v => onUpdateEntry(i, 'domain', v)}
                                />
                                <button
                                    type="button"
                                    className="diagnose-webaccess-generate-btn"
                                    onClick={async () => {
                                        try {
                                            const domain = await fetchRandomDomain(entry.domain || undefined);
                                            onUpdateEntry(i, 'domain', domain);
                                        } catch (err) {
                                            onSetError(err instanceof Error ? err.message : String(err));
                                        }
                                    }}
                                >
                                    Generate Random Subdomain
                                </button>
                            </div>
                            <div className="diagnose-webaccess-add-row">
                                <label className="diagnose-webaccess-add-label">Provider</label>
                                <select
                                    className="diagnose-webaccess-select"
                                    value={entry.provider}
                                    onChange={e => onUpdateEntry(i, 'provider', e.target.value)}
                                >
                                    <option value={DomainProviders.Cloudflare}>Cloudflare</option>
                                    <option value={DomainProviders.Ngrok}>ngrok (not supported)</option>
                                </select>
                            </div>
                            {entry.provider === DomainProviders.Cloudflare && (
                                <div className="diagnose-webaccess-add-row">
                                    <label className="diagnose-webaccess-add-label">Cloudflare Tunnel Name</label>
                                    <FlexInput
                                        inputClassName="diagnose-webaccess-input"
                                        placeholder="auto (derived from domain)"
                                        value={editTunnelName}
                                        onChange={onEditTunnelName}
                                    />
                                    <span className="diagnose-webaccess-tunnel-name-hint">
                                        Shared Cloudflare named tunnel identifier. Default: auto (derived from domain)
                                    </span>
                                </div>
                            )}
                            <div className="diagnose-webaccess-row-actions">
                                <button
                                    className="diagnose-webaccess-remove-btn"
                                    onClick={() => onRemoveEntry(i)}
                                    disabled={saving}
                                >
                                    Remove
                                </button>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {error && <div className="diagnose-security-error">{error}</div>}

            {!showAddForm ? (
                <button className="diagnose-webaccess-add-toggle" onClick={() => onSetShowAddForm(true)}>
                    + Add Domain
                </button>
            ) : (
                <div className="diagnose-webaccess-add">
                    <div className="diagnose-webaccess-add-row">
                        <label className="diagnose-webaccess-add-label">Domain</label>
                        <FlexInput
                            inputClassName="diagnose-webaccess-input"
                            placeholder="e.g. myapp.example.com"
                            value={newDomain}
                            onChange={onSetNewDomain}
                        />
                        <button
                            type="button"
                            className="diagnose-webaccess-generate-btn"
                            onClick={async () => {
                                try {
                                    const domain = await fetchRandomDomain(newDomain || undefined);
                                    onSetNewDomain(domain);
                                } catch (err) {
                                    onSetError(err instanceof Error ? err.message : String(err));
                                }
                            }}
                        >
                            Generate Random Subdomain
                        </button>
                    </div>
                    <div className="diagnose-webaccess-add-row">
                        <label className="diagnose-webaccess-add-label">Provider</label>
                        <select
                            className="diagnose-webaccess-select"
                            value={newProvider}
                            onChange={e => onSetNewProvider(e.target.value)}
                        >
                            <option value={DomainProviders.Cloudflare}>Cloudflare</option>
                            <option value={DomainProviders.Ngrok}>ngrok (not supported)</option>
                        </select>
                    </div>
                    <div className="diagnose-webaccess-add-actions">
                        <button className="diagnose-webaccess-add-btn" onClick={onAddEntry} disabled={saving || !newDomain.trim()}>
                            {saving ? 'Saving...' : 'Add Domain'}
                        </button>
                        <button className="diagnose-webaccess-cancel-btn" onClick={() => { onSetShowAddForm(false); onSetNewDomain(''); onSetNewProvider(DomainProviders.Cloudflare); }}>
                            Cancel
                        </button>
                    </div>
                </div>
            )}

            <div className="diagnose-webaccess-edit-actions">
                <button className="diagnose-webaccess-add-btn" onClick={onSave} disabled={saving}>
                    {saving ? 'Saving...' : 'Save'}
                </button>
                <button className="diagnose-webaccess-cancel-btn" onClick={onCancel}>Cancel</button>
            </div>
        </div>
    );
}
