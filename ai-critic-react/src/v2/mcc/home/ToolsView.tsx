import { useState, useEffect, useRef, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { fetchTools, fetchToolsQuick, installTool, upgradeTool } from '../../../api/tools';
import type { ToolInfo, ToolsResponse } from '../../../api/tools';
import { consumeSSEStream } from '../../../api/sse';
import { LogViewer } from '../../LogViewer';
import type { LogLine } from '../../LogViewer';
import { EffectivePathSection } from '../../../components/EffectivePathSection';
import { Loading } from '../../../pure-view/Loading';
import './ToolsView.css';

const CategoryLabels: Record<string, string> = {
    foundation: 'Foundation',
    language: 'Language',
    coding: 'Coding',
    testing: 'Testing',
    other: 'Others',
};

const CategoryOrder = ['foundation', 'language', 'coding', 'testing', 'other'] as const;

function groupToolsByCategory(tools: ToolInfo[]): { category: string; label: string; tools: ToolInfo[] }[] {
    const grouped = new Map<string, ToolInfo[]>();
    for (const tool of tools) {
        const cat = tool.category || 'other';
        const list = grouped.get(cat);
        if (list) {
            list.push(tool);
        } else {
            grouped.set(cat, [tool]);
        }
    }
    return CategoryOrder
        .filter(cat => grouped.has(cat))
        .map(cat => ({
            category: cat,
            label: CategoryLabels[cat] ?? cat,
            tools: grouped.get(cat)!,
        }));
}

export function ToolsView() {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const highlightTool = searchParams.get('tool') || '';
    const [data, setData] = useState<ToolsResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [statusLoading, setStatusLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const scrolledRef = useRef(false);

    useEffect(() => {
        if (!highlightTool || !data || scrolledRef.current) return;
        scrolledRef.current = true;
        requestAnimationFrame(() => {
            const el = document.getElementById(`tool-${highlightTool}`);
            el?.scrollIntoView({ behavior: 'smooth', block: 'center' });
        });
    }, [highlightTool, data]);

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

    const sections = useMemo(() => {
        if (!data) return [];
        return groupToolsByCategory(data.tools);
    }, [data]);

    return (
        <div className="tools-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Server Tools</h2>
            </div>

            {loading && !data ? (
                <Loading>Checking installed tools...</Loading>
            ) : data ? (
                <>
                    {error && <div className="tools-error">Error: {error}</div>}
                    <div className="tools-summary">
                        <div className="tools-summary-icon">
                            {statusLoading || checkingCount > 0 ? '⏳' : installedCount === totalCount ? '✅' : installedCount > 0 ? '⚠️' : '❌'}
                        </div>
                        <div className="tools-summary-text">
                            <span className="tools-summary-count">
                                {checkingCount > 0 ? `${checkedCount}/${totalCount}` : `${installedCount}/${totalCount}`}
                            </span>
                            <span className="tools-summary-label">
                                {checkingCount > 0
                                    ? `tools checked (${checkingCount} pending)`
                                    : 'tools installed'}
                            </span>
                        </div>
                        <div className="tools-os-badge">
                            {data.os === 'darwin' ? 'macOS' : data.os === 'linux' ? 'Linux' : data.os === 'windows' ? 'Windows' : data.os}
                        </div>
                    </div>

                    {sections.map(section => (
                        <ToolsSection
                            key={section.category}
                            label={section.label}
                            tools={section.tools}
                            os={data.os}
                            highlightTool={highlightTool}
                            onInstalled={() => loadTools(false)}
                        />
                    ))}

                    <div className="tools-path-section">
                        <EffectivePathSection />
                    </div>

                    <button className="tools-refresh-btn" onClick={() => loadTools()}>
                        {statusLoading ? 'Refreshing...' : 'Refresh'}
                    </button>
                </>
            ) : error ? (
                <div className="tools-error">Error: {error}</div>
            ) : null}
        </div>
    );
}

interface ToolsSectionProps {
    label: string;
    tools: ToolInfo[];
    os: string;
    highlightTool: string;
    onInstalled: () => void;
}

function ToolsSection({ label, tools, os, highlightTool, onInstalled }: ToolsSectionProps) {
    const hasHighlight = tools.some(t => t.name === highlightTool);
    const [expanded, setExpanded] = useState(true);

    const installedCount = tools.filter(t => t.installed).length;
    const total = tools.length;

    return (
        <div className="tools-section-group">
            <div className="tools-section-header" onClick={() => setExpanded(!expanded)}>
                <span className={`tools-section-chevron ${expanded ? 'expanded' : ''}`}>›</span>
                <h3 className="tools-section-title">{label}</h3>
                <span className="tools-section-count">{installedCount}/{total}</span>
            </div>
            {expanded && (
                <div className="tools-tools-list">
                    {tools.map(tool => (
                        <ToolCard key={tool.name} tool={tool} os={os} defaultExpanded={hasHighlight && tool.name === highlightTool} onInstalled={onInstalled} />
                    ))}
                </div>
            )}
        </div>
    );
}

interface ToolCardProps {
    tool: ToolInfo;
    os: string;
    defaultExpanded?: boolean;
    onInstalled: () => void;
}

function ToolCard({ tool, os, defaultExpanded, onInstalled }: ToolCardProps) {
    const navigate = useNavigate();
    const [expanded, setExpanded] = useState(defaultExpanded ?? false);
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
        <div id={`tool-${tool.name}`} className={`tools-tool-card ${tool.installed ? 'installed' : 'not-installed'}`}>
            <div className="tools-tool-header" onClick={() => setExpanded(!expanded)}>
                <span className="tools-tool-status">
                    {tool.checking
                        ? <span className="tools-tool-spinner" />
                        : tool.installed
                            ? '✅'
                            : installing
                                ? <span className="tools-tool-spinner" />
                                : '❌'}
                </span>
                <span className="tools-tool-name">{tool.name}</span>
                {tool.display_name && (
                    <span className="tools-tool-display-name">{tool.display_name}</span>
                )}
                {tool.installed && tool.version && (
                    <span className="tools-tool-version">{tool.version}</span>
                )}
                <div className="tools-tool-actions">
                    {!tool.checking && !tool.installed && tool.auto_install_cmd && (
                        <button
                            className="tools-tool-install-btn"
                            onClick={handleInstall}
                            disabled={installing}
                        >
                            {installing ? 'Installing...' : 'Install'}
                        </button>
                    )}
                    {tool.installed && tool.settings_path && (
                        <button
                            className="tools-tool-settings-btn"
                            onClick={(e) => { e.stopPropagation(); navigate(tool.settings_path!); }}
                        >
                            Settings
                        </button>
                    )}
                    <span className={`tools-tool-chevron ${expanded ? 'expanded' : ''}`}>›</span>
                </div>
            </div>

            {expanded && (
                <div className="tools-tool-details">
                    {tool.checking && (
                        <div className="tools-tool-row">
                            <span className="tools-tool-label">Status:</span>
                            <span className="tools-tool-value">Checking...</span>
                        </div>
                    )}
                    <div className="tools-tool-row">
                        <span className="tools-tool-label">Description:</span>
                        <span className="tools-tool-value">{tool.description}</span>
                    </div>
                    <div className="tools-tool-row">
                        <span className="tools-tool-label">Purpose:</span>
                        <span className="tools-tool-value">{tool.purpose}</span>
                    </div>
                    {tool.installed && tool.path && (
                        <div className="tools-tool-row">
                            <span className="tools-tool-label">Path:</span>
                            <code className="tools-tool-path">{tool.path}</code>
                        </div>
                    )}
                    {!tool.checking && !tool.installed && !showLogs && (
                        <div className="tools-tool-install">
                            <span className="tools-tool-install-label">Install ({os === 'darwin' ? 'macOS' : os === 'linux' ? 'Linux' : 'Windows'}):</span>
                            <code className="tools-tool-install-cmd">{getInstallCommand()}</code>
                        </div>
                    )}
                    {showLogs && (
                        <div className="tools-tool-install-logs">
                            <span className="tools-tool-install-logs-label">
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
                        <div className="tools-tool-install-all">
                            <div className="tools-tool-install-title">Installation commands:</div>
                            <div className="tools-tool-install-item">
                                <span className="tools-tool-install-os">macOS:</span>
                                <code>{tool.install_macos}</code>
                            </div>
                            <div className="tools-tool-install-item">
                                <span className="tools-tool-install-os">Linux:</span>
                                <code>{tool.install_linux}</code>
                            </div>
                            <div className="tools-tool-install-item">
                                <span className="tools-tool-install-os">Windows:</span>
                                <code>{tool.install_windows}</code>
                            </div>
                        </div>
                    )}
                    {tool.installed && (
                        <div className="tools-tool-row">
                            <button
                                className="tools-tool-upgrade-btn"
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
