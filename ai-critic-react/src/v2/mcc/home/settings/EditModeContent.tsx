import { DomainProviders, fetchRandomDomain } from '../../../../api/domains';
import type { DomainEntry } from '../../../../api/domains';
import { FlexInput } from '../../../../pure-view/FlexInput';
import { InlineError } from '../../../../pure-view/InlineError';
import { FormField } from '../../../../pure-view/form';
import { FormSelect } from '../../../../pure-view/form';
import { Button } from '../../../../pure-view/buttons/Button';
import { ButtonGroup } from '../../../../pure-view/ButtonGroup';
import '../../../../pure-view/form/FormInput.css';
import './EditModeContent.css';

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
        <>
            {editEntries.map((entry, i) => (
                <div key={i} className="edit-mode-row">
                    <FormField label="Domain">
                        <FlexInput
                            inputClassName="pure-form-input"
                            placeholder="e.g. myapp.example.com"
                            value={entry.domain}
                            onChange={v => onUpdateEntry(i, 'domain', v)}
                        />
                        <Button
                            variant="secondary"
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
                        </Button>
                    </FormField>
                    <FormField label="Provider">
                        <FormSelect
                            value={entry.provider}
                            onChange={v => onUpdateEntry(i, 'provider', v)}
                        >
                            <option value={DomainProviders.Cloudflare}>Cloudflare</option>
                            <option value={DomainProviders.Ngrok}>ngrok (not supported)</option>
                        </FormSelect>
                    </FormField>
                    {entry.provider === DomainProviders.Cloudflare && (
                        <FormField label="Cloudflare Tunnel Name">
                            <FlexInput
                                inputClassName="pure-form-input"
                                placeholder="auto (derived from domain)"
                                value={editTunnelName}
                                onChange={onEditTunnelName}
                            />
                            <span className="edit-mode-hint">
                                Shared Cloudflare named tunnel identifier. Default: auto (derived from domain)
                            </span>
                        </FormField>
                    )}
                    <div className="edit-mode-row-actions">
                        <Button
                            variant="danger"
                            onClick={() => onRemoveEntry(i)}
                            disabled={saving}
                        >
                            Remove
                        </Button>
                    </div>
                </div>
            ))}

            {error && <InlineError>{error}</InlineError>}

            {!showAddForm ? (
                <button className="edit-mode-add-toggle" onClick={() => onSetShowAddForm(true)}>
                    + Add Domain
                </button>
            ) : (
                <div className="edit-mode-add-form">
                    <FormField label="Domain">
                        <FlexInput
                            inputClassName="pure-form-input"
                            placeholder="e.g. myapp.example.com"
                            value={newDomain}
                            onChange={onSetNewDomain}
                        />
                        <Button
                            variant="secondary"
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
                        </Button>
                    </FormField>
                    <FormField label="Provider">
                        <FormSelect
                            value={newProvider}
                            onChange={onSetNewProvider}
                        >
                            <option value={DomainProviders.Cloudflare}>Cloudflare</option>
                            <option value={DomainProviders.Ngrok}>ngrok (not supported)</option>
                        </FormSelect>
                    </FormField>
                    <ButtonGroup>
                        <Button onClick={onAddEntry} disabled={saving || !newDomain.trim()}>
                            {saving ? 'Saving...' : 'Add Domain'}
                        </Button>
                        <Button variant="cancel" onClick={() => { onSetShowAddForm(false); onSetNewDomain(''); onSetNewProvider(DomainProviders.Cloudflare); }}>
                            Cancel
                        </Button>
                    </ButtonGroup>
                </div>
            )}

            <div className="edit-mode-footer">
                <Button onClick={onSave} disabled={saving}>
                    {saving ? 'Saving...' : 'Save'}
                </Button>
                <Button variant="cancel" onClick={onCancel}>Cancel</Button>
            </div>
        </>
    );
}
