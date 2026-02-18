import { useState, useRef } from 'react';
import { useFakeTerminal, type TerminalTheme } from './fake-shell/useFakeTerminal';
import { RemoteScrollbar } from '../components/remote-scrollbar/RemoteScrollbar';
import './PureTerminalMockup.css';

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
 * PureTerminalMockup - An interactive mockup showcasing the PureTerminalView component.
 * 
 * This demonstrates the core terminal view connected to a fake shell server
 * that runs entirely in the browser. Try these commands:
 * - ls, pwd, cd, echo
 * - cat readme.md
 * - tree, ps, whoami
 * - colors, help
 */
export function PureTerminalMockup() {
    const [connected, setConnected] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);
    const terminalWrapperRef = useRef<HTMLDivElement>(null);

    const {
        terminalRef,
        connected: termConnected,
        sendKey,
        reconnect,
        cols,
        rows,
        setDimensions,
        resetToFit,
    } = useFakeTerminal({
        theme: v2Theme,
        cwd: '/home/user',
        name: 'mock-shell',
    });

    // Sync connected state
    if (connected !== termConnected) {
        setConnected(termConnected);
    }

    const handleColsChange = (delta: number) => {
        const newCols = Math.max(40, Math.min(200, cols + delta));
        setDimensions(newCols, rows);
    };

    const handleRowsChange = (delta: number) => {
        const newRows = Math.max(10, Math.min(60, rows + delta));
        setDimensions(cols, newRows);
    };

    // Quick input handlers
    const handleTab = () => sendKey('\t');
    const handleArrowUp = () => sendKey('\x1b[A');
    const handleArrowDown = () => sendKey('\x1b[B');
    const handleArrowLeft = () => sendKey('\x1b[D');
    const handleArrowRight = () => sendKey('\x1b[C');
    const handleEsc = () => sendKey('\x1b');
    const handleCtrlC = () => sendKey('\x03');
    const handleCtrlL = () => sendKey('\x0c');
    const handlePaste = async () => {
        try {
            const text = await navigator.clipboard.readText();
            if (text) sendKey(text);
        } catch (err) {
            console.error('Failed to paste:', err);
        }
    };

    return (
        <div className="pure-terminal-mockup" ref={containerRef}>
            <div className="ptm-header">
                <h2>Pure Terminal View</h2>
                <p className="ptm-description">
                    Interactive terminal connected to a fake shell server running entirely in the browser.
                    Try commands: <strong>ls</strong>, <strong>pwd</strong>, <strong>help</strong>, <strong>colors</strong>
                </p>
                <div className="ptm-header-row">
                    <div className="ptm-status">
                        Status: 
                        <span className={`ptm-status-indicator ${connected ? 'connected' : 'disconnected'}`}>
                            {connected ? '● Connected (Fake Shell)' : '○ Disconnected'}
                        </span>
                        {!connected && (
                            <button className="ptm-reconnect-btn" onClick={reconnect}>
                                Reconnect
                            </button>
                        )}
                    </div>
                    <div className="ptm-dimensions">
                        <span className="ptm-dim-label">Cols:</span>
                        <button className="ptm-dim-btn" onClick={() => handleColsChange(-10)}>-10</button>
                        <button className="ptm-dim-btn" onClick={() => handleColsChange(-1)}>-</button>
                        <span className="ptm-dim-value">{cols}</span>
                        <button className="ptm-dim-btn" onClick={() => handleColsChange(1)}>+</button>
                        <button className="ptm-dim-btn" onClick={() => handleColsChange(10)}>+10</button>
                        <span className="ptm-dim-sep">|</span>
                        <span className="ptm-dim-label">Rows:</span>
                        <button className="ptm-dim-btn" onClick={() => handleRowsChange(-1)}>-</button>
                        <span className="ptm-dim-value">{rows}</span>
                        <button className="ptm-dim-btn" onClick={() => handleRowsChange(1)}>+</button>
                        <span className="ptm-dim-sep">|</span>
                        <button className="ptm-dim-btn ptm-fit-btn" onClick={resetToFit}>Fit</button>
                    </div>
                </div>
            </div>

            <div className="ptm-terminal-wrapper" ref={terminalWrapperRef}>
                <div className="ptm-terminal-container" ref={terminalRef} />
            </div>
            
            <div className="ptm-scroll-container">
                <RemoteScrollbar 
                    targetRef={terminalWrapperRef} 
                    orientation="horizontal" 
                    thickness={12}
                    trackColor="rgba(30, 41, 59, 0.3)"
                    alwaysShow={true}
                />
            </div>

            <div className="ptm-quick-input">
                <button className="ptm-quick-btn" onClick={handleTab}>Tab</button>
                <button className="ptm-quick-btn" onClick={handleArrowLeft}>←</button>
                <button className="ptm-quick-btn" onClick={handleArrowRight}>→</button>
                <button className="ptm-quick-btn" onClick={handleArrowUp}>↑</button>
                <button className="ptm-quick-btn" onClick={handleArrowDown}>↓</button>
                <button className="ptm-quick-btn" onClick={handleEsc}>Esc</button>
                <button className="ptm-quick-btn" onClick={handleCtrlC}>^C</button>
                <button className="ptm-quick-btn" onClick={handleCtrlL}>^L</button>
                <button className="ptm-quick-btn" onClick={handlePaste}>Paste</button>
            </div>
            
            <div className="ptm-info">
                <h3>Interactive Features:</h3>
                <ul>
                    <li><strong>Fully functional fake shell</strong> - Runs entirely in browser</li>
                    <li><strong>Real command processing</strong> - Try: ls, cd, pwd, echo, cat, tree, ps</li>
                    <li><strong>ANSI color support</strong> - Type 'colors' to see all colors</li>
                    <li><strong>Mobile touch scroll</strong> - Swipe to scroll terminal output</li>
                    <li><strong>Quick input buttons</strong> - For mobile convenience</li>
                </ul>
                
                <div className="ptm-commands">
                    <h4>Try these commands:</h4>
                    <code>ls</code> <code>pwd</code> <code>cd Documents</code> <code>cat readme.md</code>
                    <code>tree</code> <code>ps</code> <code>whoami</code> <code>colors</code> <code>help</code>
                </div>
            </div>
        </div>
    );
}

export default PureTerminalMockup;
