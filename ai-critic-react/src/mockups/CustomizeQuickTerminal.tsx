import { useState, useRef } from 'react';
import { useFakeTerminal } from './fake-server';
import { CustomTerminal, type CustomTerminalHandle } from '../pure-view/CustomTerminal';
import { ShortcutsBar } from '../pure-view/ShortcutsBar';
import './CustomizeQuickTerminal.css';

const DEFAULT_HISTORY = [
    'ls',
    'pwd',
    'help',
    'colors',
    'tree',
    'ps',
    'whoami',
    'date',
    'echo hello',
    'cat readme.md',
    'logo',
    'vim test.txt',
];

export function CustomizeQuickTerminal() {
    const [connected, setConnected] = useState(false);
    const [history, setHistory] = useState<string[]>(DEFAULT_HISTORY);
    const terminalContainerRef = useRef<HTMLDivElement>(null);
    const customTerminalRef = useRef<CustomTerminalHandle>(null);
    const [useXterm, setUseXterm] = useState(false);

    const {
        terminalRef,
        connected: xtermConnected,
        sendKey,
    } = useFakeTerminal({
        cwd: '/home/user',
        name: 'mock-shell',
    });

    const handleCommandExecuted = (cmd: string) => {
        setHistory(prev => {
            const filtered = prev.filter(h => h !== cmd);
            return [cmd, ...filtered].slice(0, 20);
        });
    };

    return (
        <div className="customize-quick-terminal-mockup">
            <div className="customize-quick-header">
                <h2>Customize Quick Terminal</h2>
                <p>
                    Compare the custom native terminal (optimized for iOS Safari) with xterm-based terminal.
                    The custom terminal uses simple div rendering instead of xterm.js for better mobile UX.
                </p>
            </div>

            <div className="customize-quick-toggle">
                <label>
                    <input
                        type="checkbox"
                        checked={useXterm}
                        onChange={(e) => setUseXterm(e.target.checked)}
                    />
                    <span>Use Xterm.js (instead of Custom Terminal)</span>
                </label>
            </div>

            <div className="customize-quick-container">
                <div className="v2-terminal-container">
                    <div className="v2-terminal-header">
                        <div className={`v2-terminal-status ${useXterm ? xtermConnected ? 'connected' : 'disconnected' : connected ? 'connected' : 'disconnected'}`}>
                            <span className="v2-status-dot"></span>
                            {useXterm ? (xtermConnected ? 'Connected' : 'Disconnected') : (connected ? 'Connected' : 'Disconnected')}
                        </div>
                        <span className="v2-terminal-title">bash</span>
                        <span className="v2-terminal-type">
                            {useXterm ? 'Xterm' : 'Custom'}
                        </span>
                    </div>
                    
                    {useXterm ? (
                        <div className="v2-terminal-body" ref={terminalContainerRef}>
                            <div className="v2-fake-terminal-wrapper" ref={terminalRef} />
                        </div>
                    ) : (
                        <div className="customize-quick-terminal-wrapper">
                            <CustomTerminal
                                ref={customTerminalRef}
                                cwd="/home/user"
                                name="mock-shell"
                                history={history}
                                onConnectionChange={setConnected}
                                onCommandExecuted={handleCommandExecuted}
                            />
                        </div>
                    )}
                    
                    {useXterm && (
                        <ShortcutsBar onSendKey={sendKey} />
                    )}
                </div>
            </div>

            <div className="customize-quick-features">
                <h3>Custom Terminal Features (iOS Optimized)</h3>
                <ul>
                    <li><strong>No Zoom</strong> - 16px font prevents iOS Safari zoom on input focus</li>
                    <li><strong>Native Rendering</strong> - Simple div elements instead of canvas</li>
                    <li><strong>Touch Friendly</strong> - Standard HTML input for commands</li>
                    <li><strong>Lightweight</strong> - No xterm.js dependency</li>
                    <li><strong>Fast</strong> - Simple rendering pipeline</li>
                    <li><strong>Vim Support</strong> - Alternate screen buffer for vim-like apps</li>
                </ul>
            </div>

            <div className="customize-quick-commands">
                <h3>Try These Commands</h3>
                <div className="customize-quick-command-list">
                    <code>ls</code>
                    <code>pwd</code>
                    <code>help</code>
                    <code>vim</code>
                    <code>colors</code>
                    <code>tree</code>
                    <code>ps</code>
                    <code>whoami</code>
                    <code>date</code>
                </div>
            </div>
        </div>
    );
}

export default CustomizeQuickTerminal;
