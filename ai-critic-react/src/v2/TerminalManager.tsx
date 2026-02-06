import { useState, useEffect } from 'react';
import '@xterm/xterm/css/xterm.css';
import { useTerminal } from '../hooks/useTerminal';
import type { TerminalTheme } from '../hooks/useTerminal';
import { useCurrent } from '../hooks/useCurrent';
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
}

interface TerminalManagerProps {
    isVisible: boolean;
}

// Individual terminal instance component
function TerminalInstance({ 
    id, 
    isActive,
    onConnectionChange
}: { 
    id: string; 
    isActive: boolean;
    onConnectionChange: (connected: boolean) => void;
}) {
    // Terminal is always active since parent keeps us mounted
    const { terminalRef, connected, sendKey } = useTerminal(true, { theme: v2Theme });
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

export function TerminalManager(_props: TerminalManagerProps) {
    const [tabs, setTabs] = useState<TerminalTab[]>([
        { id: 'term-1', name: 'Terminal 1' }
    ]);
    const [activeTabId, setActiveTabId] = useState('term-1');
    const [activeConnected, setActiveConnected] = useState(false);
    const tabsRef = useCurrent(tabs);

    const handleAddTab = () => {
        const newId = `term-${Date.now()}`;
        const newTabNum = tabsRef.current.length + 1;
        setTabs(prev => [...prev, { id: newId, name: `Terminal ${newTabNum}` }]);
        setActiveTabId(newId);
    };

    const handleCloseTab = (tabId: string) => {
        if (tabs.length <= 1) return; // Don't close last tab
        
        const tabIndex = tabs.findIndex(t => t.id === tabId);
        const newTabs = tabs.filter(t => t.id !== tabId);
        setTabs(newTabs);
        
        // If closing active tab, switch to adjacent tab
        if (activeTabId === tabId) {
            const newActiveIndex = Math.min(tabIndex, newTabs.length - 1);
            setActiveTabId(newTabs[newActiveIndex].id);
        }
    };

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
                        onConnectionChange={setActiveConnected}
                    />
                ))}
            </div>
        </div>
    );
}
