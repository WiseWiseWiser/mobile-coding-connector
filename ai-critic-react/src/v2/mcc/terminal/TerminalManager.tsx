import { useState, useEffect, forwardRef, useImperativeHandle } from 'react';
import '@xterm/xterm/css/xterm.css';
import { useTerminal } from '../../../hooks/useTerminal';
import type { TerminalTheme } from '../../../hooks/useTerminal';
import { useCurrent } from '../../../hooks/useCurrent';
import { fetchTerminalSessions, deleteTerminalSession } from '../../../api/terminal';
import type { TerminalSessionInfo } from '../../../api/terminal';
import './TerminalManager.css';

// Mobile-friendly dark theme
const v2Theme: TerminalTheme = {
    background: '#0f172a',
    foreground: '#e2e8f0',
    cursor: '#60a5fa',
    cursorAccent: '#0f172a',
    selectionBackground: '#334155',
    black: '#0f172a',
    red: '#ef4444',
    green: '#22c55e',
    yellow: '#eab308',
    blue: '#3b82f6',
    magenta: '#a855f7',
    cyan: '#06b6d4',
    white: '#f1f5f9',
    brightBlack: '#475569',
    brightRed: '#f87171',
    brightGreen: '#4ade80',
    brightYellow: '#facc15',
    brightBlue: '#60a5fa',
    brightMagenta: '#c084fc',
    brightCyan: '#22d3ee',
    brightWhite: '#ffffff',
};

interface TerminalTab {
    id: string;
    name: string;
    cwd?: string;
    initialCommand?: string;
    /** Backend session ID, set after session is created or when restoring */
    sessionId?: string;
}

interface TerminalManagerProps {
    isVisible: boolean;
}

/** Return the next unique "Terminal N" name that doesn't collide with existing names. */
function getNextTerminalName(existingNames: string[]): string {
    const nameSet = new Set(existingNames);
    let num = 1;
    while (nameSet.has(`Terminal ${num}`)) {
        num++;
    }
    return `Terminal ${num}`;
}

// Individual terminal instance component
function TerminalInstance({ 
    id, 
    isActive,
    cwd,
    name,
    initialCommand,
    sessionId,
    onConnectionChange,
    onSessionId,
}: { 
    id: string; 
    isActive: boolean;
    cwd?: string;
    name?: string;
    initialCommand?: string;
    sessionId?: string;
    onConnectionChange: (connected: boolean) => void;
    onSessionId: (sessionId: string) => void;
}) {
    // Terminal is always active since parent keeps us mounted
    const { terminalRef, connected, sendKey } = useTerminal(true, {
        theme: v2Theme,
        cwd,
        name,
        initialCommand,
        sessionId,
        onSessionId,
    });
    const onConnectionChangeRef = useCurrent(onConnectionChange);

    // Report connection status to parent when this instance is active
    useEffect(() => {
        if (isActive) {
            onConnectionChangeRef.current(connected);
        }
    }, [isActive, connected, onConnectionChangeRef]);

    const handleCtrlC = () => sendKey('\x03');
    const handleCtrlD = () => sendKey('\x04');
    const handleCtrlL = () => sendKey('\x0c');
    const handleTab = () => sendKey('\t');
    const handleArrowUp = () => sendKey('\x1b[A');
    const handleArrowDown = () => sendKey('\x1b[B');

    return (
        <div 
            className={`terminal-instance ${isActive ? 'active' : ''}`}
            data-terminal-id={id}
        >
            <div className="terminal-instance-content" ref={terminalRef} />
            <div className="terminal-instance-shortcuts">
                <button className="term-shortcut-btn" onClick={handleTab}>Tab</button>
                <button className="term-shortcut-btn" onClick={handleArrowUp}>↑</button>
                <button className="term-shortcut-btn" onClick={handleArrowDown}>↓</button>
                <button className="term-shortcut-btn" onClick={handleCtrlC}>Ctrl+C</button>
                <button className="term-shortcut-btn" onClick={handleCtrlD}>Ctrl+D</button>
                <button className="term-shortcut-btn" onClick={handleCtrlL}>Ctrl+L</button>
            </div>
        </div>
    );
}

export interface TerminalManagerHandle {
    /** Open a new terminal tab, optionally in a given working directory and with an initial command */
    openTab: (name: string, cwd?: string, initialCommand?: string) => void;
}

export const TerminalManager = forwardRef<TerminalManagerHandle, TerminalManagerProps>(function TerminalManager(_props, ref) {
    const [tabs, setTabs] = useState<TerminalTab[]>([]);
    const [activeTabId, setActiveTabId] = useState('');
    const [activeConnected, setActiveConnected] = useState(false);
    const [sessionsLoaded, setSessionsLoaded] = useState(false);
    const tabsRef = useCurrent(tabs);

    // Fetch existing sessions from backend on mount.
    // The `ignore` flag prevents stale async results from applying after cleanup
    // (e.g. React StrictMode double-mount or rapid tab switches).
    useEffect(() => {
        let ignore = false;

        fetchTerminalSessions()
            .then((sessions: TerminalSessionInfo[]) => {
                if (ignore) return;
                if (sessions.length === 0) {
                    const defaultTab: TerminalTab = { id: 'term-1', name: 'Terminal 1' };
                    setTabs([defaultTab]);
                    setActiveTabId(defaultTab.id);
                    setSessionsLoaded(true);
                    return;
                }
                const restoredTabs: TerminalTab[] = sessions.map(s => ({
                    id: `term-${s.id}`,
                    name: s.name,
                    cwd: s.cwd,
                    sessionId: s.id,
                }));
                setTabs(restoredTabs);
                setActiveTabId(restoredTabs[0].id);
                setSessionsLoaded(true);
            })
            .catch(() => {
                if (ignore) return;
                const defaultTab: TerminalTab = { id: 'term-1', name: 'Terminal 1' };
                setTabs([defaultTab]);
                setActiveTabId(defaultTab.id);
                setSessionsLoaded(true);
            });

        return () => { ignore = true; };
    }, []);

    const handleAddTab = () => {
        const newId = `term-${Date.now()}`;
        const newName = getNextTerminalName(tabsRef.current.map(t => t.name));
        setTabs(prev => [...prev, { id: newId, name: newName }]);
        setActiveTabId(newId);
    };

    const handleOpenTab = (name: string, cwd?: string, initialCommand?: string) => {
        const newId = `term-${Date.now()}`;
        setTabs(prev => [...prev, { id: newId, name, cwd, initialCommand }]);
        setActiveTabId(newId);
    };

    useImperativeHandle(ref, () => ({
        openTab: handleOpenTab,
    }));

    const handleCloseTab = (tabId: string) => {
        if (tabs.length <= 1) return; // Don't close last tab

        // Find the tab to get its session ID for cleanup
        const tab = tabs.find(t => t.id === tabId);
        if (tab?.sessionId) {
            deleteTerminalSession(tab.sessionId).catch(() => {});
        }
        
        const tabIndex = tabs.findIndex(t => t.id === tabId);
        const newTabs = tabs.filter(t => t.id !== tabId);
        setTabs(newTabs);
        
        // If closing active tab, switch to adjacent tab
        if (activeTabId === tabId) {
            const newActiveIndex = Math.min(tabIndex, newTabs.length - 1);
            setActiveTabId(newTabs[newActiveIndex].id);
        }
    };

    const handleSessionId = (tabId: string, sessionId: string) => {
        setTabs(prev => prev.map(t => t.id === tabId ? { ...t, sessionId } : t));
    };

    // Don't render terminals until sessions are loaded to avoid creating duplicate sessions
    if (!sessionsLoaded) {
        return (
            <div className="terminal-manager">
                <div className="terminal-manager-header">
                    <div className="terminal-tabs-container">
                        <div className="terminal-tab-item active">
                            <span className="terminal-tab-name">Loading...</span>
                        </div>
                    </div>
                    <div className="terminal-connection-status disconnected">
                        <span className="status-dot">○</span>
                        <span className="status-text">Loading</span>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="terminal-manager">
            <div className="terminal-manager-header">
                <div className="terminal-tabs-container">
                    {tabs.map(tab => (
                        <div 
                            key={tab.id}
                            className={`terminal-tab-item ${activeTabId === tab.id ? 'active' : ''}`}
                            onClick={() => setActiveTabId(tab.id)}
                        >
                            <span className="terminal-tab-name">{tab.name}</span>
                            {tabs.length > 1 && (
                                <button 
                                    className="terminal-tab-close"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        handleCloseTab(tab.id);
                                    }}
                                >
                                    ×
                                </button>
                            )}
                        </div>
                    ))}
                    <button className="terminal-tab-add" onClick={handleAddTab}>
                        +
                    </button>
                </div>
                <div className={`terminal-connection-status ${activeConnected ? 'connected' : 'disconnected'}`}>
                    <span className="status-dot">{activeConnected ? '●' : '○'}</span>
                    <span className="status-text">{activeConnected ? 'Connected' : 'Disconnected'}</span>
                </div>
            </div>
            <div className="terminal-instances-container">
                {tabs.map(tab => (
                    <TerminalInstance
                        key={tab.id}
                        id={tab.id}
                        isActive={activeTabId === tab.id}
                        cwd={tab.cwd}
                        name={tab.name}
                        initialCommand={tab.initialCommand}
                        sessionId={tab.sessionId}
                        onConnectionChange={setActiveConnected}
                        onSessionId={(sid) => handleSessionId(tab.id, sid)}
                    />
                ))}
            </div>
        </div>
    );
});
