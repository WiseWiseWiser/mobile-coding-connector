import { useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { PureTerminalView, type PureTerminalViewHandle } from '../../../components/pure-terminal/PureTerminalView';
import type { TerminalTheme } from '../../../hooks/usePureTerminal';
import './TerminalPage.css';

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

/**
 * TerminalPage - A standalone terminal page using PureTerminalView.
 * 
 * This is a simple single-terminal view without tabs, used at the /terminal route.
 * It provides a clean terminal experience with just connection status and reconnect functionality.
 */
export function TerminalPage() {
    const navigate = useNavigate();
    const terminalRef = useRef<PureTerminalViewHandle>(null);
    const [connected, setConnected] = useState(false);

    const handleBack = () => {
        navigate(-1);
    };

    return (
        <div className="terminal-page">
            <div className="terminal-page-header">
                <button className="terminal-page-back" onClick={handleBack}>
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M19 12H5M12 19l-7-7 7-7"/>
                    </svg>
                    <span>Back</span>
                </button>
                <span className="terminal-page-title">Terminal</span>
                {connected ? (
                    <div className="terminal-page-status connected">
                        <span className="status-dot">●</span>
                        <span className="status-text">Connected</span>
                    </div>
                ) : (
                    <button
                        className="terminal-page-status disconnected clickable"
                        onClick={() => terminalRef.current?.reconnect()}
                        title="Click to reconnect"
                    >
                        <span className="status-dot">○</span>
                        <span className="status-text">Reconnect</span>
                    </button>
                )}
            </div>
            <div className="terminal-page-content">
                <PureTerminalView
                    ref={terminalRef}
                    theme={v2Theme}
                    onConnectionChange={setConnected}
                    autoFocus
                />
            </div>
        </div>
    );
}

export default TerminalPage;
