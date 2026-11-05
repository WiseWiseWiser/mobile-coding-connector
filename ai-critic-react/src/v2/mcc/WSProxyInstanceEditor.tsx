import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    deleteWSInstance,
    fetchWSInstances,
    updateWSInstance,
    type WSBackendType,
    type WSProxyInstance,
} from '../../api/proxyServices';
import { WSProxyConnectPanel } from './WSProxyConnectPanel';
import { WSProxyCrumb } from './WSProxyCrumb';
import { WSProxyLogArea } from './WSProxyLogArea';
import { streamStart, streamStop, type LogLine } from './wsProxyStream';

export function WSProxyInstanceEditor({ id }: { id: string }) {
    const navigate = useNavigate();
    const [inst, setInst] = useState<WSProxyInstance | null>(null);
    const [backendType, setBackendType] = useState<WSBackendType>('self');
    const [backendURL, setBackendURL] = useState('');
    const [subdomain, setSubdomain] = useState('');
    const [autoStart, setAutoStart] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [busy, setBusy] = useState(false);
    const [pendingAction, setPendingAction] = useState<'start' | 'stop' | null>(null);
    const [logLines, setLogLines] = useState<LogLine[]>([]);
    const [step, setStep] = useState<string | null>(null);
    const [stepFailed, setStepFailed] = useState(false);
    const [logOpen, setLogOpen] = useState(false);

    useEffect(() => {
        let cancelled = false;
        fetchWSInstances().then((list) => {
            if (cancelled) return;
            const found = list.find((i) => i.id === id);
            if (!found) { setError('instance not found: ' + id); return; }
            setInst(found);
            setBackendType(found.backend_type);
            setBackendURL(found.backend_url ?? '');
            setSubdomain(found.subdomain);
            setAutoStart(found.auto_start);
        }).catch((err) => {
            if (!cancelled) setError(err instanceof Error ? err.message : String(err));
        });
        return () => { cancelled = true; };
    }, [id]);

    const reload = async () => {
        const list = await fetchWSInstances();
        const found = list.find((i) => i.id === id);
        if (found) setInst(found);
        return found;
    };

    const save = async () => {
        setBusy(true); setError(null);
        try {
            await updateWSInstance(id, {
                backend_type: backendType,
                backend_url: backendType === 'proxy' ? backendURL : '',
                subdomain,
                auto_start: autoStart,
            });
            await reload();
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally { setBusy(false); }
    };

    const toggleRun = async () => {
        const action = inst?.running ? 'stop' : 'start';
        setBusy(true); setError(null);
        setPendingAction(action);
        setLogOpen(true);
        if (action === 'start') { setLogLines([]); setStep('binary'); setStepFailed(false); }
        let failed = false;
        const onLine = (text: string, isErr?: boolean) => {
            if (isErr) failed = true;
            setLogLines((prev) => [...prev, { text, error: isErr }]);
        };
        try {
            if (action === 'start') {
                const ok = await streamStart(id, onLine, setStep);
                failed = failed || !ok;
            } else {
                await streamStop(id, onLine);
            }
        } catch (err) {
            failed = true;
            setLogLines((prev) => [...prev, { text: err instanceof Error ? err.message : String(err), error: true }]);
        } finally {
            setStepFailed(failed);
            setPendingAction(null);
            setBusy(false);
            await reload();
        }
    };

    const remove = async () => {
        setBusy(true);
        try {
            await deleteWSInstance(id);
            navigate('..', { relative: 'path' });
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally { setBusy(false); }
    };

    if (!inst) {
        return <div className="mcc-ws-page"><div className="mcc-ports-empty">{error ?? 'loading…'}</div></div>;
    }

    return (
        <div className="mcc-ws-page">
            <WSProxyCrumb parts={[{ label: 'Services', view: '../..' }, { label: 'WS Proxy', view: '..' }, { label: inst.name }]} />
            <div className="mcc-headerbar">
                <h2>Instance · {inst.name}</h2>
                <span className={`mcc-pill ${pendingAction ? 'starting' : inst.running ? 'run' : 'stop'}`}>
                    {pendingAction ? `${pendingAction}ing…` : inst.running ? 'running' : 'stopped'}
                </span>
                <span className="mcc-spacer" />
                <button className="mcc-btn sm mcc-btn-danger" onClick={remove} disabled={busy}>Delete</button>
            </div>

            {error && <div className="mcc-svc-detail-error">{error}</div>}

            <div className="mcc-grid2">
                <div className="mcc-card">
                    <div className="mcc-sect" style={{ marginTop: 0 }}>Backend(出口)</div>
                    <div className={`mcc-backend ${backendType === 'self' ? 'sel' : ''}`} onClick={() => setBackendType('self')}>
                        <span className="mcc-radio" />
                        <div><div className="mcc-bname">This server (direct)</div><div className="mcc-bdesc">本机即代理(xray freedom 出口)</div></div>
                    </div>
                    <div className={`mcc-backend ${backendType === 'proxy' ? 'sel' : ''}`} onClick={() => setBackendType('proxy')}>
                        <span className="mcc-radio" />
                        <div style={{ flex: 1 }}>
                            <div className="mcc-bname">HTTP proxy</div>
                            <input className="mcc-input" style={{ marginTop: 6 }} placeholder="http://127.0.0.1:23456"
                                value={backendURL} onChange={(e) => setBackendURL(e.target.value)}
                                onClick={(e) => { e.stopPropagation(); setBackendType('proxy'); }} />
                        </div>
                    </div>
                </div>
                <div className="mcc-card">
                    <div className="mcc-sect" style={{ marginTop: 0 }}>Instance</div>
                    <label className="mcc-lbl">子域名(owned domain)</label>
                    <input className="mcc-input" value={subdomain} onChange={(e) => setSubdomain(e.target.value)} />
                    <label className="mcc-lbl">WS path · UUID</label>
                    <input className="mcc-input" value={`${inst.ws_path} · ${inst.uuid.slice(0, 8)}…`} disabled />
                    <div className="mcc-kv" style={{ marginTop: 8 }}>
                        <span className="k">Auto start</span>
                        <span><input type="checkbox" checked={autoStart} onChange={(e) => setAutoStart(e.target.checked)} /></span>
                    </div>
                    <div className="mcc-svc-detail-actions">
                        <button className="mcc-btn mcc-btn-primary" onClick={save} disabled={busy}>Save &amp; Apply</button>
                        {inst.running
                            ? <button className="mcc-btn" onClick={toggleRun} disabled={busy}>Stop</button>
                            : <button className="mcc-btn" onClick={toggleRun} disabled={busy}>
                                {pendingAction === 'start' ? <><span className="spin" />Starting…</> : 'Start'}
                              </button>}
                    </div>
                    <div className="mcc-svc-detail-hint">域名:{inst.public_url || `${subdomain}.<domain>`}(保存后生效)</div>
                </div>
            </div>

            {(logOpen || logLines.length > 0) && (
                <WSProxyLogArea
                    id={inst.id}
                    lines={logLines}
                    streaming={pendingAction !== null}
                    step={pendingAction === 'start' || stepFailed ? step : null}
                    failed={stepFailed}
                    onClear={() => setLogLines([])}
                    onHide={() => setLogOpen(false)}
                />
            )}

            <div className="mcc-grid2" style={{ marginTop: 12 }}>
                <div className="mcc-card">
                    <WSProxyConnectPanel inst={inst} />
                </div>
                <div className="mcc-card">
                    <div className="mcc-sect" style={{ marginTop: 0 }}>CLI</div>
                    <div className="code">remote-agent ws-proxy list
remote-agent ws-proxy remove --name {inst.id}
remote-agent ws-proxy istart --name {inst.id}</div>
                </div>
            </div>
        </div>
    );
}
