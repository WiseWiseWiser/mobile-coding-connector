import { useEffect, useState } from 'react';
import { fetchWSInstanceQR, fetchWSInstanceVMess, type WSProxyInstance } from '../../api/proxyServices';

/**
 * QR + vmess + host/uuid/path for a running instance. Used on the list
 * card (compact UUID) and the editor page (full UUID).
 */
export function WSProxyConnectPanel({ inst, compact }: { inst: WSProxyInstance; compact?: boolean }) {
    const [qrUrl, setQrUrl] = useState<string | null>(null);
    const [vmess, setVmess] = useState<string | null>(null);
    const [host, setHost] = useState('');
    const [uuid, setUuid] = useState(inst.uuid);
    const [path, setPath] = useState(inst.ws_path);
    const [copied, setCopied] = useState(false);

    useEffect(() => {
        if (!inst.running) {
            setQrUrl(null);
            setVmess(null);
            return;
        }
        let cancelled = false;
        let objectUrl: string | null = null;
        fetchWSInstanceQR(inst.id).then((url) => {
            if (cancelled) {
                URL.revokeObjectURL(url);
                return;
            }
            objectUrl = url;
            setQrUrl(url);
        }).catch(() => {
            if (!cancelled) setQrUrl(null);
        });
        fetchWSInstanceVMess(inst.id).then((r) => {
            if (cancelled) return;
            setVmess(r.vmess_link);
            if (r.host) setHost(r.host);
            if (r.uuid) setUuid(r.uuid);
            if (r.path) setPath(r.path);
        }).catch(() => {
            if (!cancelled) setVmess(null);
        });
        return () => {
            cancelled = true;
            if (objectUrl) URL.revokeObjectURL(objectUrl);
        };
    }, [inst.id, inst.running, inst.public_url, inst.uuid, inst.ws_path]);

    if (!inst.running) {
        return compact ? null : <div className="mcc-ports-empty">QR 不可用(实例未运行)</div>;
    }

    const hostLabel = host || inst.public_url?.replace(/^https?:\/\//, '') || `${inst.subdomain}.<domain>`;
    const uuidLabel = compact && uuid.length > 12 ? `${uuid.slice(0, 8)}…` : uuid;

    const copy = async () => {
        if (!vmess) return;
        await navigator.clipboard?.writeText(vmess);
        setCopied(true);
        window.setTimeout(() => setCopied(false), 900);
    };

    return (
        <div className="mcc-connect" onClick={(e) => e.stopPropagation()}>
            <div className="mcc-connect-qr">
                {qrUrl
                    ? <img className="mcc-qr-img" src={qrUrl} alt="vmess QR" />
                    : <div className="mcc-ports-empty">loading QR…</div>}
                <div className="mcc-connect-cap">Shadowrocket / v2rayN 扫码导入</div>
            </div>
            <div className="mcc-connect-fields">
                <div className="mcc-sect" style={{ marginTop: 0 }}>Connect</div>
                <div className="mcc-kv"><span className="k">Host</span><span className="v mcc-mono">{hostLabel}</span></div>
                <div className="mcc-kv"><span className="k">Port · TLS</span><span className="v mcc-mono">443 · tls / ws</span></div>
                <div className="mcc-kv"><span className="k">UUID</span><span className="v mcc-mono">{uuidLabel}</span></div>
                <div className="mcc-kv"><span className="k">Path</span><span className="v mcc-mono">{path || '/ws'}</span></div>
                {vmess && (
                    <div className="mcc-vmess-row">
                        <code>{vmess}</code>
                        <button className="mcc-btn sm mcc-btn-primary" onClick={copy}>{copied ? 'Copied' : 'Copy'}</button>
                    </div>
                )}
                {compact && (
                    <div className="mcc-svc-detail-hint">手机扫左侧二维码即可走本机内网。Edit 可改 backend / 子域名。</div>
                )}
            </div>
        </div>
    );
}
