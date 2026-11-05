import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
    createWSInstance,
    deleteWSInstance,
    fetchWSInstances,
    type WSBackendType,
    type WSProxyInstance,
} from '../../api/proxyServices';
import { WSProxyConnectPanel } from './WSProxyConnectPanel';
import { WSProxyCrumb } from './WSProxyCrumb';
import { WSProxyInstanceEditor } from './WSProxyInstanceEditor';
import { WSProxyLogArea } from './WSProxyLogArea';
import { streamStart, streamStop, type LogLine } from './wsProxyStream';

/** WSProxyPage is the sub-page for the multi-instance WS Proxy system service. */
export function WSProxyPage() {
    const { instanceId } = useParams();
    if (instanceId) {
        return <WSProxyInstanceEditor id={instanceId} />;
    }
    return <WSProxyList />;
}

function WSProxyList() {
    const [instances, setInstances] = useState<WSProxyInstance[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [showAdd, setShowAdd] = useState(false);
    const navigate = useNavigate();

    const [pendingId, setPendingId] = useState<string | null>(null);
    const [pendingAction, setPendingAction] = useState<'start' | 'stop' | null>(null);
    const [logFor, setLogFor] = useState<string | null>(null);
    const [logLines, setLogLines] = useState<LogLine[]>([]);
    const [step, setStep] = useState<string | null>(null);
    const [stepFailed, setStepFailed] = useState(false);
    const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

    useEffect(() => {
        let cancelled = false;
        fetchWSInstances().then((list) => {
            if (!cancelled) { setInstances(list); setError(null); }
        }).catch((err) => {
            if (!cancelled) setError(err instanceof Error ? err.message : String(err));
        });
        return () => { cancelled = true; };
    }, []);

    const refresh = async () => {
        try {
            setInstances(await fetchWSInstances());
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const runOp = async (inst: WSProxyInstance, action: 'start' | 'stop') => {
        setPendingId(inst.id);
        setPendingAction(action);
        setLogFor(inst.id);
        setLogLines([]);
        setStep(action === 'start' ? 'binary' : null);
        setStepFailed(false);
        let failed = false;
        const onLine = (text: string, isErr?: boolean) => {
            if (isErr) failed = true;
            setLogLines((prev) => [...prev, { text, error: isErr }]);
        };
        try {
            if (action === 'start') {
                const ok = await streamStart(inst.id, onLine, setStep);
                failed = failed || !ok;
            } else {
                await streamStop(inst.id, onLine);
            }
        } catch (err) {
            failed = true;
            setLogLines((prev) => [...prev, { text: err instanceof Error ? err.message : String(err), error: true }]);
        } finally {
            setStepFailed(failed);
            setPendingId(null);
            setPendingAction(null);
            await refresh();
        }
    };

    const removeInstance = async (inst: WSProxyInstance) => {
        setPendingId(inst.id);
        setPendingAction('stop');
        setLogFor(inst.id);
        try {
            if (inst.running) {
                await streamStop(inst.id, (text, isErr) => {
                    setLogLines((prev) => [...prev, { text, error: isErr }]);
                });
            }
            await deleteWSInstance(inst.id);
            setConfirmDeleteId(null);
            await refresh();
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally {
            setPendingId(null);
            setPendingAction(null);
        }
    };

    const running = instances.filter((i) => i.running).length;

    return (
        <div className="mcc-ws-page">
            <WSProxyCrumb parts={[{ label: 'Services', view: '..' }, { label: 'WS Proxy' }]} />
            <div className="mcc-headerbar">
                <h2>WS Proxy</h2>
                <span className={`mcc-pill ${running > 0 ? 'run' : 'stop'}`}>{running}/{instances.length} running</span>
                <span className="mcc-spacer" />
                <button className="mcc-btn mcc-btn-primary" onClick={() => setShowAdd((v) => !v)}>+ Add Instance</button>
            </div>
            <div className="mcc-sub">
                Xray VMess + WebSocket over Cloudflare Tunnel。每实例 = 独立域名 + UUID + 后端出口(self 直连本机内网,或 HTTP proxy 指向本机/远端端口)。
            </div>
            {error && <div className="mcc-svc-detail-error">{error}</div>}

            {showAdd && (
                <AddInstanceForm
                    onDone={(created) => { setShowAdd(false); if (created) navigate(created.id); void refresh(); }}
                />
            )}

            <div className="mcc-sect">Instances</div>
            {instances.map((inst) => {
                const pending = pendingId === inst.id;
                return (
                    <div key={inst.id} className="mcc-inst-row" style={pending ? { borderColor: 'rgba(251,191,36,.5)' } : undefined}>
                        <div className="mcc-card-top">
                            <div>
                                <div className="mcc-name">
                                    <a className="mcc-crumb-link" onClick={() => navigate(inst.id)}>{inst.name}</a>
                                    <span className={`mcc-pill ${pending ? 'starting' : inst.running ? 'run' : 'stop'}`} style={{ marginLeft: 8 }}>
                                        {pending ? (pendingAction === 'start' ? 'starting…' : 'stopping…') : inst.running ? 'running' : 'stopped'}
                                    </span>
                                    {inst.legacy && <span className="mcc-pill stop" style={{ marginLeft: 4 }}>migrated</span>}
                                </div>
                                <div className="mcc-meta">
                                    {inst.public_url || `ws://${inst.subdomain} (not published)`}
                                    {inst.running ? ` · :${inst.effective_port ?? inst.listen_port}` : ''}
                                </div>
                            </div>
                            <span className="mcc-spacer" />
                            <span className={`mcc-bchip ${inst.backend_type === 'self' ? 'self' : 'http'}`}>
                                {inst.backend_type === 'self' ? 'This server · direct' : `HTTP proxy · ${inst.backend_url}`}
                            </span>
                            <div className="mcc-inst-actions">
                                <button className="mcc-btn sm" disabled={pendingId !== null} onClick={() => navigate(inst.id)}>Edit</button>
                                {inst.running
                                    ? <button className="mcc-btn sm" disabled={pendingId !== null} onClick={() => void runOp(inst, 'stop')}>Stop</button>
                                    : <button className="mcc-btn sm mcc-btn-primary" disabled={pendingId !== null} onClick={() => void runOp(inst, 'start')}>
                                        {pending ? <><span className="spin" />Starting…</> : 'Start'}
                                    </button>}
                                <button className="mcc-btn sm mcc-btn-danger" disabled={pendingId !== null} onClick={() => setConfirmDeleteId(inst.id)}>Delete</button>
                            </div>
                        </div>
                        {confirmDeleteId === inst.id && (
                            <div className="mcc-del-confirm">
                                删除实例 <b>{inst.name}</b>? 会先 Stop,再清掉域名 mapping。此操作不可恢复。
                                <div className="mcc-del-confirm-row">
                                    <button className="mcc-btn sm mcc-btn-danger" disabled={pendingId !== null} onClick={() => void removeInstance(inst)}>Delete</button>
                                    <button className="mcc-btn sm" disabled={pendingId !== null} onClick={() => setConfirmDeleteId(null)}>Cancel</button>
                                </div>
                            </div>
                        )}
                        {inst.running && <WSProxyConnectPanel inst={inst} compact />}
                        {logFor === inst.id && (
                            <WSProxyLogArea
                                id={inst.id}
                                lines={logLines}
                                streaming={pending}
                                step={pendingAction === 'start' || stepFailed ? step : null}
                                failed={stepFailed}
                                onClear={() => setLogLines([])}
                                onHide={() => setLogFor(null)}
                            />
                        )}
                    </div>
                );
            })}
            {!instances.length && !error && (
                <div className="mcc-ports-empty">No instances yet — add one above.</div>
            )}
        </div>
    );
}

function AddInstanceForm({ onDone }: { onDone: (created: WSProxyInstance | null) => void }) {
    const [name, setName] = useState('');
    const [backendType, setBackendType] = useState<WSBackendType>('self');
    const [backendURL, setBackendURL] = useState('');
    const [subdomain, setSubdomain] = useState('');
    const [autoStart, setAutoStart] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [busy, setBusy] = useState(false);

    const submit = async () => {
        setBusy(true); setError(null);
        try {
            const created = await createWSInstance({
                name: name || subdomain,
                backend_type: backendType,
                backend_url: backendType === 'proxy' ? backendURL : undefined,
                subdomain,
                auto_start: autoStart,
            });
            onDone(created);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally { setBusy(false); }
    };

    return (
        <div className="mcc-card">
            <div className="mcc-sect" style={{ marginTop: 0 }}>New instance</div>
            {error && <div className="mcc-svc-detail-error">{error}</div>}
            <label className="mcc-lbl">名称</label>
            <input className="mcc-input" value={name} onChange={(e) => setName(e.target.value)} placeholder="留空则用子域名" />
            <label className="mcc-lbl">子域名(owned domain)</label>
            <input className="mcc-input" value={subdomain} onChange={(e) => setSubdomain(e.target.value)} placeholder="ws-intranet" />

            <label className="mcc-lbl">Backend(出口)</label>
            <div className={`mcc-backend ${backendType === 'self' ? 'sel' : ''}`} onClick={() => setBackendType('self')}>
                <span className="mcc-radio" />
                <div>
                    <div className="mcc-bname">This server (direct)</div>
                    <div className="mcc-bdesc">ai-critic 本机直接访问内网 — 本机即代理,无需 upstream</div>
                </div>
            </div>
            <div className={`mcc-backend ${backendType === 'proxy' ? 'sel' : ''}`} onClick={() => setBackendType('proxy')}>
                <span className="mcc-radio" />
                <div style={{ flex: 1 }}>
                    <div className="mcc-bname">HTTP proxy</div>
                    <input
                        className="mcc-input"
                        style={{ marginTop: 6 }}
                        placeholder="http://127.0.0.1:23456 或 http://10.53.64.69:23456"
                        value={backendURL}
                        onChange={(e) => setBackendURL(e.target.value)}
                        onClick={(e) => { e.stopPropagation(); setBackendType('proxy'); }}
                    />
                </div>
            </div>

            <div className="mcc-kv" style={{ marginTop: 10 }}>
                <span className="k">Auto start</span>
                <span>
                    <input type="checkbox" checked={autoStart} onChange={(e) => setAutoStart(e.target.checked)} />
                </span>
            </div>
            <div className="mcc-svc-detail-actions">
                <button className="mcc-btn mcc-btn-primary" onClick={submit} disabled={busy || !subdomain}>Create</button>
                <button className="mcc-btn" onClick={() => onDone(null)}>Cancel</button>
            </div>
        </div>
    );
}
