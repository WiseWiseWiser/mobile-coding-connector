import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchTools, fetchToolsQuick, installTool, upgradeTool } from '../../../api/tools';
import type { ToolInfo, ToolsResponse } from '../../../api/tools';
import { consumeSSEStream } from '../../../api/sse';
import { LogViewer } from '../../LogViewer';
import type { LogLine } from '../../LogViewer';
import { EffectivePathSection } from '../../../components/EffectivePathSection';
import './DiagnoseView.css';

export function DiagnoseView() {
    const navigate = useNavigate();
    const [data, setData] = useState<ToolsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [statusLoading, setStatusLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const loadTools = async (showLoading = true) => {
        setError(null);

        if (showLoading) {
            if (!data) {
                setLoading(true);
            }
            try {
                const quickData = await fetchToolsQuick();
                setData(quickData);
                setLoading(false);
            } catch {
                // Ignore quick-fetch errors and continue with full status fetch.
            }
        }

        setStatusLoading(true);
        try {
            const fullData = await fetchTools();
            setData(fullData);
        } catch (err) {
            const message = err instanceof Error ? err.message : String(err);
            setError(message);
        } finally {
            setStatusLoading(false);
            setLoading(false);
        }
    };

    useEffect(() => {
        loadTools();
    }, []);

    const installedCount = data?.tools.filter(t => t.installed).length ?? 0;
    const totalCount = data?.tools.length ?? 0;
    const checkingCount = data?.tools.filter(t => t.checking).length ?? 0;
    const checkedCount = totalCount - checkingCount;

    return (
        <div className="diagnose-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Server Tools</h2>
            </div>

            {loading && !data ? (
                <div className="diagnose-loading">Checking installed tools...</div>
            ) : data ? (
                <>
                    {error && <div className="diagnose-error">Error: {error}</div>}
                    <div className="diagnose-summary">
                        <div className="diagnose-summary-icon">
                            {statusLoading || checkingCount > 0 ? '⏳' : installedCount === totalCount ? '✅' : installedCount > 0 ? '⚠️' : '❌'}
                        </div>
                        <div className="diagnose-summary-text">
                            <span className="diagnose-summary-count">
                                {checkingCount > 0 ? `${checkedCount}/${totalCount}` : `${installedCount}/${totalCount}`}
                            </span>
                            <span className="diagnose-summary-label">
                                {checkingCount > 0
                                    ? `tools checked (${checkingCount} pending)`
                                    : 'tools installed'}
                            </span>
                        </div>
                        <div className="diagnose-os-badge">
                            {data.os === 'darwin' ? 'macOS' : data.os === 'linux' ? 'Linux' : data.os === 'windows' ? 'Windows' : data.os}
                        </div>
                    </div>

                    <div className="diagnose-tools-list">
                        {data.tools.map(tool => (
                            <ToolCard key={tool.name} tool={tool} os={data.os} onInstalled={() => loadTools(false)} />
                        ))}
                    </div>

                    <div className="diagnose-path-section">
                        <EffectivePathSection />
                    </div>

                    <button className="diagnose-refresh-btn" onClick={() => loadTools()}>
                        {statusLoading ? 'Refreshing...' : 'Refresh'}
                    </button>
                </>
            ) : error ? (
                <div className="diagnose-error">Error: {error}</div>
            ) : null}
        </div>
    );
}

interface ToolCardProps {
    tool: ToolInfo;
    os: string;
    onInstalled: () => void;
}

function ToolCard({ tool, os, onInstalled }: ToolCardProps) {
    const navigate = useNavigate();
    const [expanded, setExpanded] = useState(false);
    const [installing, setInstalling] = useState(false);
    const [installLogs, setInstallLogs] = useState<LogLine[]>([]);
    const [installDone, setInstallDone] = useState(false);
    const [installError, setInstallError] = useState(false);
    const [upgrading, setUpgrading] = useState(false);

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

    const getUpgradeCommand = () => {
        switch (os) {
            case 'darwin':
                return tool.upgrade_macos;
            case 'linux':
                return tool.upgrade_linux;
            case 'windows':
                return tool.upgrade_windows;
            default:
                return tool.upgrade_linux;
        }
    };

    const hasUpgradeCommand = () => {
        const cmd = getUpgradeCommand();
        return cmd && cmd.trim().length > 0;
    };

    const handleInstall = async (e: React.MouseEvent) => {
        e.stopPropagation();
        setInstalling(true);
        setInstallLogs([]);
        setInstallDone(false);
        setInstallError(false);
        setExpanded(true);

        try {
            const resp = await installTool(tool.name);
            await consumeSSEStream(resp, {
                onLog: (line) => setInstallLogs(prev => [...prev, line]),
                onError: (line) => {
                    setInstallLogs(prev => [...prev, line]);
                    setInstallError(true);
                },
                onDone: (message) => {
                    setInstallLogs(prev => [...prev, { text: message }]);
                    setInstallDone(true);
                    onInstalled();
                },
            });
        } catch (err) {
            setInstallLogs(prev => [...prev, { text: String(err), error: true }]);
            setInstallError(true);
        }
        setInstalling(false);
    };

    const handleUpgrade = async (e: React.MouseEvent) => {
        e.stopPropagation();
        if (!hasUpgradeCommand()) return;
        
        setUpgrading(true);
        setInstallLogs([]);
        setInstallDone(false);
        setInstallError(false);
        setExpanded(true);

        try {
            const resp = await upgradeTool(tool.name);
            await consumeSSEStream(resp, {
                onLog: (line) => setInstallLogs(prev => [...prev, line]),
                onError: (line) => {
                    setInstallLogs(prev => [...prev, line]);
                    setInstallError(true);
                },
                onDone: (message) => {
                    setInstallLogs(prev => [...prev, { text: message }]);
                    setInstallDone(true);
                    onInstalled();
                },
            });
        } catch (err) {
            setInstallLogs(prev => [...prev, { text: String(err), error: true }]);
            setInstallError(true);
        }
        setUpgrading(false);
    };

    const showLogs = installLogs.length > 0;

    return (
        <div className={`diagnose-tool-card ${tool.installed ? 'installed' : 'not-installed'}`}>
            <div className="diagnose-tool-header" onClick={() => setExpanded(!expanded)}>
                <span className="diagnose-tool-status">
                    {tool.checking
                        ? <span className="diagnose-tool-spinner" />
                        : tool.installed
                            ? '✅'
                            : installing
                                ? <span className="diagnose-tool-spinner" />
                                : '❌'}
                </span>
                <span className="diagnose-tool-name">{tool.name}</span>
                {tool.installed && tool.version && (
                    <span className="diagnose-tool-version">{tool.version}</span>
                )}
                {!tool.checking && !tool.installed && tool.auto_install_cmd && (
                    <button
                        className="diagnose-tool-install-btn"
                        onClick={handleInstall}
                        disabled={installing}
                    >
                        {installing ? 'Installing...' : 'Install'}
                    </button>
                )}
                {tool.installed && tool.settings_path && (
                    <button
                        className="diagnose-tool-settings-btn"
                        onClick={(e) => { e.stopPropagation(); navigate(tool.settings_path!); }}
                    >
                        Settings
                    </button>
                )}
                <span className={`diagnose-tool-chevron ${expanded ? 'expanded' : ''}`}>›</span>
            </div>

            {expanded && (
                <div className="diagnose-tool-details">
                    {tool.checking && (
                        <div className="diagnose-tool-row">
                            <span className="diagnose-tool-label">Status:</span>
                            <span className="diagnose-tool-value">Checking...</span>
                        </div>
                    )}
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
                    {!tool.checking && !tool.installed && !showLogs && (
                        <div className="diagnose-tool-install">
                            <span className="diagnose-tool-install-label">Install ({os === 'darwin' ? 'macOS' : os === 'linux' ? 'Linux' : 'Windows'}):</span>
                            <code className="diagnose-tool-install-cmd">{getInstallCommand()}</code>
                        </div>
                    )}
                    {showLogs && (
                        <div className="diagnose-tool-install-logs">
                            <span className="diagnose-tool-install-logs-label">
                                {installing ? 'Installing...' : installDone ? 'Installed successfully' : installError ? 'Installation failed' : ''}
                            </span>
                            <LogViewer
                                lines={installLogs}
                                pending={installing}
                                pendingMessage="Installing..."
                                maxHeight={200}
                            />
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
                    {tool.installed && (
                        <div className="diagnose-tool-row">
                            <button
                                className="diagnose-tool-upgrade-btn"
                                onClick={handleUpgrade}
                                disabled={upgrading || !hasUpgradeCommand()}
                                title={hasUpgradeCommand() ? 'Upgrade to latest version' : 'Upgrade not available for this tool'}
                                style={{ marginTop: '8px' }}
                            >
                                {upgrading ? 'Upgrading...' : hasUpgradeCommand() ? 'Upgrade' : 'Upgrade (not available)'}
                            </button>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
