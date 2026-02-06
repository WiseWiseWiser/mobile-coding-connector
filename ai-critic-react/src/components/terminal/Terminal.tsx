import '@xterm/xterm/css/xterm.css';
import { useTerminal } from '../../hooks/useTerminal';
import './Terminal.css';

interface TerminalProps {
    isOpen: boolean;
    onClose: () => void;
}

export function Terminal({ isOpen, onClose }: TerminalProps) {
    const { terminalRef, connected } = useTerminal(isOpen);

    if (!isOpen) {
        return null;
    }

    return (
        <div className="terminal-overlay">
            <div className="terminal-container">
                <div className="terminal-header">
                    <span className="terminal-title">Terminal</span>
                    <span className={`terminal-status ${connected ? 'connected' : 'disconnected'}`}>
                        {connected ? '● Connected' : '○ Disconnected'}
                    </span>
                    <button className="terminal-close" onClick={onClose}>×</button>
                </div>
                <div className="terminal-content" ref={terminalRef} />
            </div>
        </div>
    );
}
