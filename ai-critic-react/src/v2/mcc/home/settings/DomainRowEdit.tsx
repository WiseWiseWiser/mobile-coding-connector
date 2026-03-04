import { useState } from 'react';
import { DomainProviders, fetchRandomDomain } from '../../../../api/domains';
import type { DomainEntry } from '../../../../api/domains';
import { FlexInput } from '../../../../pure-view/FlexInput';
import { FormField } from '../../../../pure-view/form';
import { FormSelect } from '../../../../pure-view/form';
import { Button } from '../../../../pure-view/buttons';
import '../../../../pure-view/form/FormInput.css';
import './DomainRowEdit.css';

export interface DomainRowEditProps {
    entry: DomainEntry;
    tunnelName: string;
    onSave: (entry: DomainEntry, tunnelName: string) => void;
    onRemove: () => void;
    onCancel: () => void;
    onError: (err: string | null) => void;
}

export function DomainRowEdit({ entry, tunnelName, onSave, onRemove, onCancel, onError }: DomainRowEditProps) {
    const [domain, setDomain] = useState(entry.domain);
    const [provider, setProvider] = useState(entry.provider);
    const [editTunnelName, setEditTunnelName] = useState(tunnelName);

    const handleSave = () => {
        if (!domain.trim()) return;
        onSave({ domain: domain.trim(), provider }, editTunnelName);
    };

    return (
        <div className="domain-row-edit">
            <FormField label="Domain">
                <FlexInput
                    inputClassName="pure-form-input"
                    placeholder="e.g. myapp.example.com"
                    value={domain}
                    onChange={setDomain}
                />
                <Button
                    variant="secondary"
                    onClick={async () => {
                        try {
                            const d = await fetchRandomDomain(domain || undefined);
                            setDomain(d);
                        } catch (err) {
                            onError(err instanceof Error ? err.message : String(err));
                        }
                    }}
                >
                    Generate Random Subdomain
                </Button>
            </FormField>
            <FormField label="Provider">
                <FormSelect value={provider} onChange={setProvider}>
                    <option value={DomainProviders.Cloudflare}>Cloudflare</option>
                    <option value={DomainProviders.Ngrok}>ngrok (not supported)</option>
                </FormSelect>
            </FormField>
            {provider === DomainProviders.Cloudflare && (
                <FormField label="Cloudflare Tunnel Name">
                    <FlexInput
                        inputClassName="pure-form-input"
                        placeholder="auto (derived from domain)"
                        value={editTunnelName}
                        onChange={setEditTunnelName}
                    />
                    <span className="domain-row-edit-hint">
                        Shared Cloudflare named tunnel identifier. Default: auto (derived from domain)
                    </span>
                </FormField>
            )}
            <div className="domain-row-edit-actions">
                <Button onClick={handleSave} disabled={!domain.trim()}>
                    Save
                </Button>
                <Button variant="cancel" onClick={onCancel}>Cancel</Button>
                <Button variant="danger" onClick={onRemove}>Remove</Button>
            </div>
        </div>
    );
}
