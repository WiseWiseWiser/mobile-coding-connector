import { useState, useEffect, forwardRef, useImperativeHandle, useCallback } from 'react';
import '@xterm/xterm/css/xterm.css';
import { useTerminal } from '../../../hooks/useTerminal';
import type { TerminalTheme } from '../../../hooks/useTerminal';
import { useCurrent } from '../../../hooks/useCurrent';
import { deleteTerminalSession } from '../../../api/terminal';
import { useV2Context } from '../../V2Context';
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

export interface TerminalTab {
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

    const [ctrlPressed, setCtrlPressed] = useState(false);

    const handleSendKey = useCallback((key: string) => {
        if (ctrlPressed) {
            // Convert to control character when ctrl is pressed
            // Ctrl+A = 0x01, Ctrl+B = 0x02, etc.
            const charCode = key.charCodeAt(0);
            if (charCode >= 97 && charCode <= 122) {
                // lowercase a-z
                sendKey(String.fromCharCode(charCode - 96));
            } else if (charCode >= 65 && charCode <= 90) {
                // uppercase A-Z
                sendKey(String.fromCharCode(charCode - 64));
            } else {
                sendKey(key);
            }
            setCtrlPressed(false);
        } else {
            sendKey(key);
        }
    }, [ctrlPressed, sendKey]);

    const handleCtrl = () => setCtrlPressed(true);
    const handleCtrlC = () => sendKey('\x03');
    const handleCtrlA = () => sendKey('\x01');
    const handleCtrlR = () => sendKey('\x12');
    const handleCtrlL = () => sendKey('\x0c');
    const handleTab = () => handleSendKey('\t');
    const handleArrowUp = () => sendKey('\x1b[A');
    const handleArrowDown = () => sendKey('\x1b[B');
    const handleArrowLeft = () => sendKey('\x1b[D');
    const handleArrowRight = () => sendKey('\x1b[C');
    const handlePaste = async () => {
        try {
            const text = await navigator.clipboard.readText();
            if (text) {
                handleSendKey(text);
            }
        } catch (err) {
            console.error('Failed to paste from clipboard:', err);
        }
    };

    return (
        <div 
            className={`terminal-instance ${isActive ? 'active' : ''}`}
            data-terminal-id={id}
        >
            <div className="terminal-instance-content" ref={terminalRef} />
            <div className="terminal-instance-shortcuts">
                <button className={`term-shortcut-btn ${ctrlPressed ? 'active' : ''}`} onClick={handleCtrl}>Ctrl</button>
                <button className="term-shortcut-btn" onClick={handleTab}>Tab</button>
                <button className="term-shortcut-btn" onClick={handleArrowLeft}>←</button>
                <button className="term-shortcut-btn" onClick={handleArrowRight}>→</button>
                <button className="term-shortcut-btn" onClick={handleArrowUp}>↑</button>
                <button className="term-shortcut-btn" onClick={handleArrowDown}>↓</button>
                <button className="term-shortcut-btn" onClick={handleCtrlC}>Ctrl+C</button>
                <button className="term-shortcut-btn" onClick={handleCtrlA}>Ctrl+A</button>
                <button className="term-shortcut-btn" onClick={handleCtrlR}>Ctrl+R</button>
                <button className="term-shortcut-btn" onClick={handleCtrlL}>Ctrl+L</button>
                <button className="term-shortcut-btn" onClick={handlePaste}>Paste</button>
            </div>
        </div>
    );
}

export interface TerminalManagerHandle {
    /** Open a new terminal tab, optionally in a given working directory and with an initial command */
    openTab: (name: string, cwd?: string, initialCommand?: string) => void;
}

export const TerminalManager = forwardRef<TerminalManagerHandle, TerminalManagerProps>(function TerminalManager(_props, ref) {
    const {
        terminalTabs,
        setTerminalTabs,
        activeTerminalTabId,
        setActiveTerminalTabId,
        terminalSessionsLoaded,
    } = useV2Context();
    
    const [activeConnected, setActiveConnected] = useState(false);

    const handleAddTab = () => {
        const newId = `term-${Date.now()}`;
        const newName = getNextTerminalName(terminalTabs.map(t => t.name));
        setTerminalTabs([...terminalTabs, { id: newId, name: newName }]);
        setActiveTerminalTabId(newId);
    };

    const handleOpenTab = (name: string, cwd?: string, initialCommand?: string) => {
        const newId = `term-${Date.now()}`;
        setTerminalTabs([...terminalTabs, { id: newId, name, cwd, initialCommand }]);
        setActiveTerminalTabId(newId);
    };

    useImperativeHandle(ref, () => ({
        openTab: handleOpenTab,
    }));

    const handleCloseTab = (tabId: string) => {
        if (terminalTabs.length <= 1) return; // Don't close last tab

        // Find the tab to get its session ID for cleanup
        const tab = terminalTabs.find(t => t.id === tabId);
        if (tab?.sessionId) {
            deleteTerminalSession(tab.sessionId).catch(() => {});
        }
        
        const tabIndex = terminalTabs.findIndex(t => t.id === tabId);
        const newTabs = terminalTabs.filter(t => t.id !== tabId);
        setTerminalTabs(newTabs);
        
        // If closing active tab, switch to adjacent tab
        if (activeTerminalTabId === tabId) {
            const newActiveIndex = Math.min(tabIndex, newTabs.length - 1);
            setActiveTerminalTabId(newTabs[newActiveIndex].id);
        }
    };

    const handleSessionId = (tabId: string, sessionId: string) => {
        setTerminalTabs(terminalTabs.map(t => t.id === tabId ? { ...t, sessionId } : t));
    };

    // Don't render terminals until sessions are loaded to avoid creating duplicate sessions
    if (!terminalSessionsLoaded) {
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
                    {terminalTabs.map(tab => (
                        <div 
                            key={tab.id}
                            className={`terminal-tab-item ${activeTerminalTabId === tab.id ? 'active' : ''}`}
                            onClick={() => setActiveTerminalTabId(tab.id)}
                        >
                            <span className="terminal-tab-name">{tab.name}</span>
                            {terminalTabs.length > 1 && (
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
                {terminalTabs.map(tab => (
                    <TerminalInstance
                        key={tab.id}
                        id={tab.id}
                        isActive={activeTerminalTabId === tab.id}
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
