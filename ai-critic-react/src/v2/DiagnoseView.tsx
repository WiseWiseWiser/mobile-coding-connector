import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchTools } from '../api/tools';
import type { ToolInfo, ToolsResponse } from '../api/tools';
import './DiagnoseView.css';

export function DiagnoseView() {
    const navigate = useNavigate();
    const [data, setData] = useState<ToolsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const loadTools = () => {
        setLoading(true);
        setError(null);
        fetchTools()
            .then(d => { setData(d); setLoading(false); })
            .catch(err => { setError(err.message); setLoading(false); });
    };

    useEffect(() => {
        loadTools();
    }, []);

    const installedCount = data?.tools.filter(t => t.installed).length ?? 0;
    const totalCount = data?.tools.length ?? 0;

    return (
        <div className="diagnose-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>System Diagnostics</h2>
            </div>

            {loading ? (
                <div className="diagnose-loading">Checking installed tools...</div>
            ) : error ? (
                <div className="diagnose-error">Error: {error}</div>
            ) : data ? (
                <>
                    <div className="diagnose-summary">
                        <div className="diagnose-summary-icon">
                            {installedCount === totalCount ? '✅' : installedCount > 0 ? '⚠️' : '❌'}
                        </div>
                        <div className="diagnose-summary-text">
                            <span className="diagnose-summary-count">{installedCount}/{totalCount}</span>
                            <span className="diagnose-summary-label">tools installed</span>
                        </div>
                        <div className="diagnose-os-badge">
                            {data.os === 'darwin' ? 'macOS' : data.os === 'linux' ? 'Linux' : data.os === 'windows' ? 'Windows' : data.os}
                        </div>
                    </div>

                    <div className="diagnose-tools-list">
                        {data.tools.map(tool => (
                            <ToolCard key={tool.name} tool={tool} os={data.os} />
                        ))}
                    </div>

                    <button className="diagnose-refresh-btn" onClick={loadTools}>
                        Refresh
                    </button>
                </>
            ) : null}
        </div>
    );
}

interface ToolCardProps {
    tool: ToolInfo;
    os: string;
}

function ToolCard({ tool, os }: ToolCardProps) {
    const [expanded, setExpanded] = useState(false);

    const getInstallCommand = () => {
        switch (os) {
            case 'darwin':
                return tool.install_macos;
            case 'linux':
                return tool.install_linux;
            case 'windows':
                return tool.install_windows;
            default:
                return tool.install_linux;
        }
    };

    return (
        <div className={`diagnose-tool-card ${tool.installed ? 'installed' : 'not-installed'}`}>
            <div className="diagnose-tool-header" onClick={() => setExpanded(!expanded)}>
                <span className="diagnose-tool-status">
                    {tool.installed ? '✅' : '❌'}
                </span>
                <span className="diagnose-tool-name">{tool.name}</span>
                {tool.installed && tool.version && (
                    <span className="diagnose-tool-version">{tool.version}</span>
                )}
                <span className={`diagnose-tool-chevron ${expanded ? 'expanded' : ''}`}>›</span>
            </div>

            {expanded && (
                <div className="diagnose-tool-details">
                    <div className="diagnose-tool-row">
                        <span className="diagnose-tool-label">Description:</span>
                        <span className="diagnose-tool-value">{tool.description}</span>
                    </div>
                    <div className="diagnose-tool-row">
                        <span className="diagnose-tool-label">Purpose:</span>
                        <span className="diagnose-tool-value">{tool.purpose}</span>
                    </div>
                    {tool.installed && tool.path && (
                        <div className="diagnose-tool-row">
                            <span className="diagnose-tool-label">Path:</span>
                            <code className="diagnose-tool-path">{tool.path}</code>
                        </div>
                    )}
                    {!tool.installed && (
                        <div className="diagnose-tool-install">
                            <span className="diagnose-tool-install-label">Install ({os === 'darwin' ? 'macOS' : os === 'linux' ? 'Linux' : 'Windows'}):</span>
                            <code className="diagnose-tool-install-cmd">{getInstallCommand()}</code>
                        </div>
                    )}
                    {tool.installed && (
                        <div className="diagnose-tool-install-all">
                            <div className="diagnose-tool-install-title">Installation commands:</div>
                            <div className="diagnose-tool-install-item">
                                <span className="diagnose-tool-install-os">macOS:</span>
                                <code>{tool.install_macos}</code>
                            </div>
                            <div className="diagnose-tool-install-item">
                                <span className="diagnose-tool-install-os">Linux:</span>
                                <code>{tool.install_linux}</code>
                            </div>
                            <div className="diagnose-tool-install-item">
                                <span className="diagnose-tool-install-os">Windows:</span>
                                <code>{tool.install_windows}</code>
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
