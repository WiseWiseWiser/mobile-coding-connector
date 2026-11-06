import { ConfirmModal } from '../ConfirmModal';
import { formatBytes, killHostProcess, type HostProcess } from '../../../api/processes';

export function SystemKillModal({
    proc,
    onClose,
    onKilled,
}: {
    proc: HostProcess;
    onClose: () => void;
    onKilled: () => void;
}) {
    const locked = !!proc.protected;
    const isTunnel = (proc.tags ?? []).includes('tun');
    const warning = locked
        ? proc.protected
        : isTunnel
            ? 'This looks like the Cloudflare connector. Killing it disconnects this page until the tunnel respawns.'
            : undefined;

    return (
        <ConfirmModal
            title="Kill Process"
            message={locked ? 'This process is protected and cannot be killed from the UI.' : 'Are you sure you want to kill this process?'}
            warning={warning}
            info={{
                Name: proc.name,
                PID: String(proc.pid),
                User: proc.user,
                'CPU / RSS': `${proc.cpu_percent.toFixed(1)}% · ${formatBytes(proc.rss_bytes)}`,
            }}
            command={locked ? `kill ${proc.pid}` : `kill -TERM ${proc.pid}`}
            confirmLabel={locked ? 'Cannot Kill' : 'Kill Process'}
            confirmVariant={locked ? 'default' : 'danger'}
            onConfirm={async () => {
                if (locked) return;
                await killHostProcess(proc.pid, 'TERM');
                onKilled();
            }}
            onClose={onClose}
        />
    );
}
