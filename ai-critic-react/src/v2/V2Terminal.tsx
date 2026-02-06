import '@xterm/xterm/css/xterm.css';
import { useTerminal } from '../hooks/useTerminal';
import './V2Terminal.css';

// Mobile-friendly dark theme
const v2Theme = {
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

interface V2TerminalProps {
    className?: string;
}

export function V2Terminal({ className }: V2TerminalProps) {
    const { terminalRef, connected, sendKey } = useTerminal(true, { theme: v2Theme });

    const handleCtrlC = () => sendKey('\x03');
    const handleCtrlD = () => sendKey('\x04');
    const handleCtrlL = () => sendKey('\x0c');
    const handleTab = () => sendKey('\t');
    const handleArrowUp = () => sendKey('\x1b[A');
    const handleArrowDown = () => sendKey('\x1b[B');

    return (
        <div className={`v2-terminal ${className || ''}`}>
            <div className="v2-terminal-header">
                <div className="v2-terminal-tabs">
                    <button className="v2-terminal-tab active">Terminal</button>
                </div>
                <span className={`v2-terminal-status ${connected ? 'connected' : 'disconnected'}`}>
                    {connected ? '● Connected' : '○ Disconnected'}
                </span>
            </div>
            <div className="v2-terminal-content" ref={terminalRef} />
            <div className="v2-terminal-shortcuts">
                <button className="v2-shortcut-btn" onClick={handleTab}>Tab</button>
                <button className="v2-shortcut-btn" onClick={handleArrowUp}>↑</button>
                <button className="v2-shortcut-btn" onClick={handleArrowDown}>↓</button>
                <button className="v2-shortcut-btn" onClick={handleCtrlC}>Ctrl+C</button>
                <button className="v2-shortcut-btn" onClick={handleCtrlD}>Ctrl+D</button>
                <button className="v2-shortcut-btn" onClick={handleCtrlL}>Ctrl+L</button>
            </div>
        </div>
    );
}
