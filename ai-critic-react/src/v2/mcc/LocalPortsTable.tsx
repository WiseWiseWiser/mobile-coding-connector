import { useState } from 'react';
import type { LocalPortInfo } from '../../api/ports';
import { PlusIcon } from '../icons';
import './LocalPortsTable.css';

const SortFields = {
    Port: 'port',
    Pid: 'pid',
    Command: 'command',
} as const;

type SortField = typeof SortFields[keyof typeof SortFields];

const SortDirections = {
    Asc: 'asc',
    Desc: 'desc',
} as const;

type SortDirection = typeof SortDirections[keyof typeof SortDirections];

export interface LocalPortsTableProps {
    ports: LocalPortInfo[];
    loading: boolean;
    error: string | null;
    forwardedPorts: Set<number>;
    onForwardPort: (port: number) => void;
}

export function LocalPortsTable({ 
    ports, 
    loading, 
    error, 
    forwardedPorts,
    onForwardPort 
}: LocalPortsTableProps) {
    const [sortField, setSortField] = useState<SortField>(SortFields.Port);
    const [sortDirection, setSortDirection] = useState<SortDirection>(SortDirections.Asc);

    const handleSort = (field: SortField) => {
        if (sortField === field) {
            setSortDirection(sortDirection === SortDirections.Asc ? SortDirections.Desc : SortDirections.Asc);
        } else {
            setSortField(field);
            setSortDirection(SortDirections.Asc);
        }
    };

    const sortedPorts = [...ports].sort((a, b) => {
        let comparison = 0;
        switch (sortField) {
            case SortFields.Port:
                comparison = a.port - b.port;
                break;
            case SortFields.Pid:
                comparison = a.pid - b.pid;
                break;
            case SortFields.Command:
                comparison = a.command.localeCompare(b.command);
                break;
        }
        return sortDirection === SortDirections.Asc ? comparison : -comparison;
    });

    const getSortIndicator = (field: SortField) => {
        if (sortField !== field) return '⇅';
        return sortDirection === SortDirections.Asc ? '↑' : '↓';
    };

    if (loading && ports.length === 0) {
        return (
            <div className="mcc-lp-loading">
                <div className="mcc-loading-spinner" />
                <span>Loading local ports...</span>
            </div>
        );
    }

    if (error && ports.length === 0) {
        return (
            <div className="mcc-lp-error">
                <span>{error}</span>
            </div>
        );
    }

    return (
        <div className="mcc-lp-section">
            <div className="mcc-lp-header">
                <h3 className="mcc-lp-title">Local Listening Ports</h3>
                <span className="mcc-lp-count">{ports.length}</span>
            </div>
            
            {ports.length === 0 ? (
                <div className="mcc-lp-empty">
                    No listening ports found on this machine.
                </div>
            ) : (
                <>
                    <div className="mcc-lp-sort-bar">
                        <button className={`mcc-lp-sort-btn ${sortField === SortFields.Port ? 'active' : ''}`} onClick={() => handleSort(SortFields.Port)}>
                            Port {getSortIndicator(SortFields.Port)}
                        </button>
                        <button className={`mcc-lp-sort-btn ${sortField === SortFields.Pid ? 'active' : ''}`} onClick={() => handleSort(SortFields.Pid)}>
                            PID {getSortIndicator(SortFields.Pid)}
                        </button>
                        <button className={`mcc-lp-sort-btn ${sortField === SortFields.Command ? 'active' : ''}`} onClick={() => handleSort(SortFields.Command)}>
                            Process {getSortIndicator(SortFields.Command)}
                        </button>
                    </div>
                    <div className="mcc-lp-list">
                        {sortedPorts.map((port) => {
                            const isForwarded = forwardedPorts.has(port.port);
                            return (
                                <div key={`${port.port}-${port.pid}`} className="mcc-lp-row">
                                    <div className="mcc-lp-row-main">
                                        <code className="mcc-lp-port-num">{port.port}</code>
                                        <span className="mcc-lp-command">{port.command}</span>
                                        {isForwarded ? (
                                            <span className="mcc-lp-forwarded-badge">Forwarded</span>
                                        ) : (
                                            <button 
                                                className="mcc-lp-forward-btn"
                                                onClick={() => onForwardPort(port.port)}
                                                title={`Forward port ${port.port}`}
                                            >
                                                <PlusIcon />
                                            </button>
                                        )}
                                    </div>
                                    <div className="mcc-lp-row-meta">
                                        <span className="mcc-lp-pid">PID {port.pid}</span>
                                        <span className="mcc-lp-ppid">PPID {port.ppid}</span>
                                    </div>
                                    {port.cmdline && (
                                        <div className="mcc-lp-cmdline">{port.cmdline}</div>
                                    )}
                                </div>
                            );
                        })}
                    </div>
                </>
            )}
        </div>
    );
}
