import { useState } from 'react';
import { DomainProviders, fetchRandomDomain } from '../../../../api/domains';
import type { DomainEntry } from '../../../../api/domains';
import { FlexInput } from '../../../../pure-view/FlexInput';
import { FormField } from '../../../../pure-view/form';
import { FormSelect } from '../../../../pure-view/form';
import { Button } from '../../../../pure-view/buttons';
import { ButtonGroup } from '../../../../pure-view/ButtonGroup';
import '../../../../pure-view/form/FormInput.css';
import './AddDomainForm.css';

export interface AddDomainFormProps {
    onAdd: (entry: DomainEntry) => void;
    onCancel: () => void;
    onError: (err: string | null) => void;
}

export function AddDomainForm({ onAdd, onCancel, onError }: AddDomainFormProps) {
    const [domain, setDomain] = useState('');
    const [provider, setProvider] = useState<string>(DomainProviders.Cloudflare);

    const handleAdd = () => {
        if (!domain.trim()) return;
        onAdd({ domain: domain.trim(), provider });
    };

    return (
        <div className="add-domain-form">
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
            <ButtonGroup>
                <Button onClick={handleAdd} disabled={!domain.trim()}>
                    Add Domain
                </Button>
                <Button variant="cancel" onClick={onCancel}>
                    Cancel
                </Button>
            </ButtonGroup>
        </div>
    );
}
