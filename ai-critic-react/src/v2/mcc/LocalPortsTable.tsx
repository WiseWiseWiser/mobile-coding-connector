import { useState } from 'react';
import type { LocalPortInfo } from '../../api/ports';
import { PlusIcon } from '../icons';

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
    const [sortField, setSortField] = useState<'port' | 'pid' | 'command'>('port');
    const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

    const handleSort = (field: 'port' | 'pid' | 'command') => {
        if (sortField === field) {
            setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
        } else {
            setSortField(field);
            setSortDirection('asc');
        }
    };

    const sortedPorts = [...ports].sort((a, b) => {
        let comparison = 0;
        switch (sortField) {
            case 'port':
                comparison = a.port - b.port;
                break;
            case 'pid':
                comparison = a.pid - b.pid;
                break;
            case 'command':
                comparison = a.command.localeCompare(b.command);
                break;
        }
        return sortDirection === 'asc' ? comparison : -comparison;
    });

    const getSortIndicator = (field: 'port' | 'pid' | 'command') => {
        if (sortField !== field) return '⇅';
        return sortDirection === 'asc' ? '↑' : '↓';
    };

    if (loading && ports.length === 0) {
        return (
            <div className="mcc-local-ports-loading">
                <div className="mcc-loading-spinner" />
                <span>Loading local ports...</span>
            </div>
        );
    }

    if (error && ports.length === 0) {
        return (
            <div className="mcc-local-ports-error">
                <span>⚠️ {error}</span>
            </div>
        );
    }

    return (
        <div className="mcc-local-ports-section">
            <div className="mcc-local-ports-header">
                <h3>Local Listening Ports</h3>
                <span className="mcc-local-ports-count">{ports.length} ports</span>
            </div>
            
            {ports.length === 0 ? (
                <div className="mcc-local-ports-empty">
                    No listening ports found on this machine.
                </div>
            ) : (
                <div className="mcc-local-ports-table-container">
                    <table className="mcc-local-ports-table">
                        <thead>
                            <tr>
                                <th 
                                    className="mcc-local-ports-sortable" 
                                    onClick={() => handleSort('port')}
                                >
                                    Port {getSortIndicator('port')}
                                </th>
                                <th 
                                    className="mcc-local-ports-sortable" 
                                    onClick={() => handleSort('pid')}
                                >
                                    PID {getSortIndicator('pid')}
                                </th>
                                <th 
                                    className="mcc-local-ports-sortable" 
                                    onClick={() => handleSort('command')}
                                >
                                    Process {getSortIndicator('command')}
                                </th>
                                <th>Action</th>
                            </tr>
                        </thead>
                        <tbody>
                            {sortedPorts.map((port) => {
                                const isForwarded = forwardedPorts.has(port.port);
                                return (
                                    <tr key={`${port.port}-${port.pid}`}>
                                        <td className="mcc-local-port-cell">
                                            <code className="mcc-port-number">{port.port}</code>
                                        </td>
                                        <td className="mcc-local-port-cell">
                                            <code>{port.pid}</code>
                                        </td>
                                        <td className="mcc-local-port-cell">
                                            <span className="mcc-command-name">{port.command}</span>
                                        </td>
                                        <td className="mcc-local-port-cell">
                                            {isForwarded ? (
                                                <span className="mcc-port-forwarded-badge">Forwarded</span>
                                            ) : (
                                                <button 
                                                    className="mcc-forward-port-btn"
                                                    onClick={() => onForwardPort(port.port)}
                                                    title={`Forward port ${port.port}`}
                                                >
                                                    <PlusIcon />
                                                    <span>Forward</span>
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}
