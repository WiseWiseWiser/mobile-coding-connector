import { useState } from 'react';
import { updatePort, type OriginTarget } from '../../api/ports';
import type { PortForward } from '../../hooks/usePortForwards';
import { PortTargetFields } from './PortTargetFields';

interface Props { port: PortForward; onClose: () => void }

export function PortTargetEditor({ port, onClose }: Props) {
    const [host, setHost] = useState(port.target?.host || '127.0.0.1');
    const [scheme, setScheme] = useState<OriginTarget['scheme']>(port.target?.scheme || 'http');
    const [number, setNumber] = useState(String(port.target?.port || port.localPort));
    const [hostname, setHostname] = useState(port.label);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    async function save() {
        if (!port.id) return;
        setSaving(true); setError('');
        try {
            const result = await updatePort(port.id, { target: { host, scheme, port: Number(number) }, label: hostname });
            if (result.origin && !result.origin.tcpReachable) {
                setError(`Saved; origin unreachable: ${result.origin.error}. Tunnel status does not prove target health.`);
            } else { onClose(); }
        } catch (e) { setError(e instanceof Error ? e.message : String(e)); }
        finally { setSaving(false); }
    }
    return <div className="mcc-add-port-form" aria-label="Edit mapping target">
        <h3>Edit mapping <code>{port.id}</code></h3>
        <div className="mcc-form-field"><label>Public hostname / label</label><input value={hostname} onChange={e => setHostname(e.target.value)} /></div>
        <div className="mcc-form-field"><label>Target port</label><input type="number" min="1" max="65535" value={number} onChange={e => setNumber(e.target.value)} /></div>
        <PortTargetFields host={host} scheme={scheme} port={number} onHostChange={setHost} onSchemeChange={setScheme} />
        <p>Authentication and provider remain unchanged. Saving closes existing upgraded connections; clients must reconnect.</p>
        {error && <div className="mcc-ports-error" role="alert">{error}</div>}
        <div className="mcc-port-actions"><button className="mcc-port-action-btn" disabled={saving} onClick={save}>{saving ? 'Saving…' : 'Save mapping'}</button><button className="mcc-port-action-btn" disabled={saving} onClick={onClose}>Close</button></div>
    </div>;
}
