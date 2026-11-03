import './PortTargetFields.css';
import { useId, useState } from 'react';
import { checkPortTarget, type OriginTarget, type PortTargetCheck } from '../../api/ports';

interface Props {
    host: string;
    scheme: 'http' | 'https';
    port: string;
    onHostChange: (host: string) => void;
    onSchemeChange: (scheme: 'http' | 'https') => void;
}

export function PortTargetFields({ host, scheme, port, onHostChange, onSchemeChange }: Props) {
    const inputId = useId();
    const [check, setCheck] = useState<{ key: string; result?: PortTargetCheck; error?: string } | null>(null);
    const [checking, setChecking] = useState(false);
    const key = JSON.stringify([host, scheme, port]);
    const current = check?.key === key ? check : null;
    const target: OriginTarget = { host, scheme, port: Number(port) };
    const normalizedHost = host.replace(/^\[|\]$/g, '');
    const authority = normalizedHost.includes(':') ? `[${normalizedHost.replace(/%/g, '%25')}]` : normalizedHost;
    async function diagnose() {
        setChecking(true);
        try { setCheck({ key, result: await checkPortTarget(target) }); }
        catch (error) { setCheck({ key, error: error instanceof Error ? error.message : String(error) }); }
        finally { setChecking(false); }
    }
    return <div className="mcc-target-fields">
        <div className="mcc-form-field">
            <label htmlFor={`${inputId}-host`}>Target host · resolved by this server</label>
            <input id={`${inputId}-host`} value={host} onChange={e => onHostChange(e.target.value)} placeholder="127.0.0.1 / 192.168.1.1 / macbook.local / fe80::1%eth0" />
        </div>
        <div className="mcc-form-field">
            <label htmlFor={`${inputId}-scheme`}>Target protocol</label>
            <select id={`${inputId}-scheme`} value={scheme} onChange={e => onSchemeChange(e.target.value as Props['scheme'])}>
                <option value="http">HTTP</option><option value="https">HTTPS (certificate verified)</option>
            </select>
        </div>
        <div className="mcc-target-preview"><code>{scheme}://{authority || '127.0.0.1'}:{port || '<port>'}</code></div>
        <p className="mcc-ports-help-text">IPv6 zones name an interface on this server (Pi: %eth0, not Mac: %en7). The origin must listen on a reachable address, not only its localhost.</p>
        <p className="mcc-ports-help-text">HTTP connection pooling · WebSocket upgrades · SSE streaming. Host / Origin are preserved; configure the target application's trusted hosts separately.</p>
        <button type="button" className="mcc-port-action-btn" disabled={checking || !Number(port)} onClick={diagnose}>{checking ? 'Checking from server…' : 'Check target from server'}</button>
        {current && <div role="status" className={current.result?.tcpReachable ? 'mcc-target-check-ok' : 'mcc-ports-error'}>
            {current.error || (current.result?.tcpReachable ? 'TCP reachable — not an application health or WebSocket check.' : `Origin unreachable: ${current.result?.error}`)}
        </div>}
    </div>;
}
