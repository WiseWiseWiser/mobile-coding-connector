import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
    createWSInstance,
    deleteWSInstance,
    fetchWSInstanceQR,
    fetchWSInstanceVMess,
    fetchWSInstances,
    startWSInstance,
    stopWSInstance,
    updateWSInstance,
    type WSBackendType,
    type WSProxyInstance,
} from '../../api/proxyServices';

export interface PortsOutletCtx {
    navigateToView: (view: string) => void;
    [key: string]: unknown;
}

function Crumb({ parts }: { parts: { label: string; view?: string }[] }) {
    const navigate = useNavigate();
    return (
        <div className="mcc-crumb">
            {parts.map((p, i) => (
                <span key={i}>
                    {i > 0 && <span className="mcc-crumb-sep">/</span>}
                    {p.view !== undefined
                        ? <a className="mcc-crumb-link" onClick={() => navigate(p.view!)}>{p.label}</a>
                        : <span>{p.label}</span>}
                </span>
            ))}
        </div>
    );
}

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
    const [, setBusy] = useState(false);
    const [showAdd, setShowAdd] = useState(false);
    const navigate = useNavigate();

    const refresh = useCallback(async () => {
        try {
            setInstances(await fetchWSInstances());
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
    }, []);

    useEffect(() => { void refresh(); }, [refresh]);

    const act = async (fn: () => Promise<unknown>) => {
        setBusy(true);
        try { await fn(); await refresh(); } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally { setBusy(false); }
    };

    const running = instances.filter((i) => i.running).length;

    return (
        <div className="mcc-ws-page">
            <Crumb parts={[{ label: 'Services', view: '..' }, { label: 'WS Proxy' }]} />
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
            {instances.map((inst) => (
                <div key={inst.id} className="mcc-inst-row" onClick={() => navigate(inst.id)}>
                    <div className="mcc-card-top">
                        <div>
                            <div className="mcc-name">
                                {inst.name}
                                <span className={`mcc-pill ${inst.running ? 'run' : 'stop'}`} style={{ marginLeft: 8 }}>
                                    {inst.running ? 'running' : 'stopped'}
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
                        {inst.running
                            ? <button className="mcc-btn sm" onClick={(e) => { e.stopPropagation(); void act(() => stopWSInstance(inst.id)); }}>Stop</button>
                            : <button className="mcc-btn sm mcc-btn-primary" onClick={(e) => { e.stopPropagation(); void act(() => startWSInstance(inst.id)); }}>Start</button>}
                    </div>
                </div>
            ))}
            {!instances.length && !error && (
                <div className="mcc-ports-empty">No instances yet — add one above.</div>
            )}

            <div className="mcc-warnbox">
                💡 RSP 场景:在 RSP 的 ai-critic 里加一个实例,后端选 HTTP proxy → <code>http://&lt;Mac-IP&gt;:23456</code>(线缆直达),
                手机扫 RSP 实例的二维码即可用到 Mac 的内网。CONNECT 直连/灰云走 HTTP Proxy 服务;CF 域名走本页的 WS Proxy。
            </div>
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

function WSProxyInstanceEditor({ id }: { id: string }) {
    const [inst, setInst] = useState<WSProxyInstance | null>(null);
    const [backendType, setBackendType] = useState<WSBackendType>('self');
    const [backendURL, setBackendURL] = useState('');
    const [subdomain, setSubdomain] = useState('');
    const [autoStart, setAutoStart] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [busy, setBusy] = useState(false);
    const [qrUrl, setQrUrl] = useState<string | null>(null);
    const [vmess, setVmess] = useState<string | null>(null);

    useEffect(() => {
        fetchWSInstances().then((list) => {
            const found = list.find((i) => i.id === id);
            if (!found) { setError('instance not found: ' + id); return; }
            setInst(found);
            setBackendType(found.backend_type);
            setBackendURL(found.backend_url ?? '');
            setSubdomain(found.subdomain);
            setAutoStart(found.auto_start);
        }).catch((err) => setError(err instanceof Error ? err.message : String(err)));
    }, [id]);

    const refreshQR = useCallback(() => {
        fetchWSInstanceQR(id).then(setQrUrl).catch(() => setQrUrl(null));
        fetchWSInstanceVMess(id).then((r) => setVmess(r.vmess_link)).catch(() => setVmess(null));
    }, [id]);

    useEffect(() => { refreshQR(); }, [refreshQR]);

    const save = async () => {
        setBusy(true); setError(null);
        try {
            await updateWSInstance(id, {
                backend_type: backendType,
                backend_url: backendType === 'proxy' ? backendURL : '',
                subdomain,
                auto_start: autoStart,
            });
            const list = await fetchWSInstances();
            const found = list.find((i) => i.id === id);
            setInst(found ?? null);
            refreshQR();
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally { setBusy(false); }
    };

    const toggleRun = async () => {
        setBusy(true); setError(null);
        try {
            const updated = inst?.running ? await stopWSInstance(id) : await startWSInstance(id);
            setInst(updated);
            refreshQR();
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        } finally { setBusy(false); }
    };

    const remove = async () => {
        setBusy(true);
        try { await deleteWSInstance(id); } finally { setBusy(false); }
    };

    if (!inst) {
        return <div className="mcc-ws-page"><div className="mcc-ports-empty">{error ?? 'loading…'}</div></div>;
    }

    return (
        <div className="mcc-ws-page">
            <Crumb parts={[{ label: 'Services', view: '../..' }, { label: 'WS Proxy', view: '..' }, { label: inst.name }]} />
            <div className="mcc-headerbar">
                <h2>Instance · {inst.name}</h2>
                <span className={`mcc-pill ${inst.running ? 'run' : 'stop'}`}>{inst.running ? 'running' : 'stopped'}</span>
                <span className="mcc-spacer" />
                {!inst.legacy && <button className="mcc-btn danger sm" onClick={remove} disabled={busy}>Delete</button>}
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
                            : <button className="mcc-btn" onClick={toggleRun} disabled={busy}>Start</button>}
                    </div>
                    <div className="mcc-svc-detail-hint">域名:{inst.public_url || `${subdomain}.<domain>`}(保存后生效)</div>
                </div>
            </div>

            <div className="mcc-grid2" style={{ marginTop: 12 }}>
                <div className="mcc-card">
                    <div className="mcc-sect" style={{ marginTop: 0 }}>QR(Shadowrocket 扫码导入)</div>
                    {qrUrl
                        ? <img className="mcc-qr-img" src={qrUrl} alt="vmess QR" style={{ margin: '0 auto', display: 'block' }} />
                        : <div className="mcc-ports-empty">QR 不可用(实例未运行)</div>}
                    {vmess && (
                        <div className="mcc-mono mcc-vmess-link">
                            {vmess.slice(0, 42)}…
                            <button className="mcc-btn" onClick={() => navigator.clipboard?.writeText(vmess)}>Copy</button>
                        </div>
                    )}
                </div>
                <div className="mcc-card">
                    <div className="sect">CLI</div>
                    <div className="code">remote-agent ws-proxy list
remote-agent ws-proxy remove --name {inst.id}
remote-agent ws-proxy istart --name {inst.id}</div>
                </div>
            </div>
        </div>
    );
}
