import { useState } from 'react';
import { killProcess } from '../../api/ports';
import { ConfirmModal } from './ConfirmModal';
import type { LocalPortInfo } from './ConfirmModal';

export interface KillProcessModalProps {
    port: LocalPortInfo;
    protectedPorts: number[];
    onClose: () => void;
    onKilled: () => void;
}

export function KillProcessModal({ port, protectedPorts, onClose, onKilled }: KillProcessModalProps) {
    const isPidOne = port.pid === 1;
    const isProtected = protectedPorts.includes(port.port);
    const canKill = !isPidOne && !isProtected;

    const command = `kill ${port.pid}`;

    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleConfirm = async () => {
        if (!canKill) return;
        setLoading(true);
        setError(null);
        try {
            await killProcess(port.pid, port.port);
            onKilled();
        } catch (err) {
            setError(String(err));
            setLoading(false);
        }
    };

    if (!canKill) {
        return (
            <ConfirmModal
                title="Kill Process"
                message={isPidOne ? 'Cannot kill init process (PID 1).' : 'This port is protected and cannot be killed.'}
                info={{
                    Port: String(port.port),
                    PID: String(port.pid),
                    Command: port.command,
                }}
                command={command}
                confirmLabel="Cannot Kill"
                confirmVariant="default"
                onConfirm={async () => {}}
                onClose={onClose}
                warning={isPidOne ? 'Cannot kill init process (PID 1).' : 'This port is protected and cannot be killed.'}
            />
        );
    }

    return (
        <ConfirmModal
            title="Kill Process"
            message="Are you sure you want to kill this process?"
            info={{
                Port: String(port.port),
                PID: String(port.pid),
                Command: port.command,
            }}
            command={command}
            confirmLabel="Kill Process"
            confirmVariant="danger"
            onConfirm={handleConfirm}
            onClose={onClose}
            loading={loading}
            error={error}
        />
    );
}
