import '@xterm/xterm/css/xterm.css';
import { useState, useCallback, useEffect, useRef } from 'react';
import { useTerminal } from '../../../hooks/useTerminal';
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
    const { terminalRef, connected, sendKey, reconnect } = useTerminal(true, { theme: v2Theme });
    const [ctrlPressed, setCtrlPressed] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);
    const wasDisconnectedRef = useRef(false);

    // Auto-reconnect when terminal regains focus if disconnected
    useEffect(() => {
        const handleFocus = () => {
            if (!connected && wasDisconnectedRef.current) {
                reconnect();
            }
            wasDisconnectedRef.current = !connected;
        };

        const handleVisibilityChange = () => {
            if (!document.hidden && !connected && wasDisconnectedRef.current) {
                reconnect();
            }
            wasDisconnectedRef.current = !connected;
        };

        const container = containerRef.current;
        container?.addEventListener('focus', handleFocus);
        document.addEventListener('visibilitychange', handleVisibilityChange);

        return () => {
            container?.removeEventListener('focus', handleFocus);
            document.removeEventListener('visibilitychange', handleVisibilityChange);
        };
    }, [connected, reconnect]);

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
    const handleCtrlL = () => sendKey('\x0c');
    const handleTab = () => handleSendKey('\t');
    const handleArrowUp = () => sendKey('\x1b[A');
    const handleArrowDown = () => sendKey('\x1b[B');
    const handleArrowLeft = () => sendKey('\x1b[D');
    const handleArrowRight = () => sendKey('\x1b[C');

    return (
        <div className={`v2-terminal ${className || ''}`} ref={containerRef} tabIndex={0}>
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
                <button className={`v2-shortcut-btn ${ctrlPressed ? 'active' : ''}`} onClick={handleCtrl}>Ctrl</button>
                <button className="v2-shortcut-btn" onClick={handleTab}>Tab</button>
                <button className="v2-shortcut-btn" onClick={handleArrowLeft}>←</button>
                <button className="v2-shortcut-btn" onClick={handleArrowRight}>→</button>
                <button className="v2-shortcut-btn" onClick={handleArrowUp}>↑</button>
                <button className="v2-shortcut-btn" onClick={handleArrowDown}>↓</button>
                <button className="v2-shortcut-btn" onClick={handleCtrlC}>Ctrl+C</button>
                <button className="v2-shortcut-btn" onClick={handleCtrlA}>Ctrl+A</button>
                <button className="v2-shortcut-btn" onClick={handleCtrlL}>Ctrl+L</button>
            </div>
        </div>
    );
}
