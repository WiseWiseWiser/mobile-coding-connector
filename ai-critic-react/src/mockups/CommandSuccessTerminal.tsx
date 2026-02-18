import { useRef, useState } from 'react';
import '@xterm/xterm/css/xterm.css';
import { useFakeTerminal } from './fake-server';
import { ShortcutsBar } from '../pure-view/ShortcutsBar';
import './CommandSuccessTerminal.css';

export function XtermQuickTerminal() {
    const terminalContainerRef = useRef<HTMLDivElement>(null);
    const [command, setCommand] = useState('');

    const {
        terminalRef,
        connected,
        sendKey,
    } = useFakeTerminal({
        cwd: '/home/user',
        name: 'mock-shell',
    });

    const handleSubmit = (cmd: string) => {
        sendKey(cmd + '\r');
    };

    const handleRun = () => {
        if (command.trim()) {
            sendKey(command + '\r');
            setCommand('');
        }
    };

    return (
        <div className="command-success-mockup">
            <div className="command-success-header">
                <h2>Command Success Terminal</h2>
                <p>
                    Interactive terminal connected to a fake bash server running in the browser.
                    Try commands like <code>ls</code>, <code>pwd</code>, <code>help</code>, <code>colors</code>, 
                    or any shell command.
                </p>
            </div>

            <div className="command-success-container">
                <div className="v2-terminal-container">
                    <div className="v2-terminal-header">
                        <div className={`v2-terminal-status ${connected ? 'connected' : 'disconnected'}`}>
                            <span className="v2-status-dot"></span>
                            {connected ? 'Connected' : 'Disconnected'}
                        </div>
                        <span className="v2-terminal-title">bash</span>
                    </div>
                    <div className="v2-terminal-body" ref={terminalContainerRef}>
                        <div className="v2-fake-terminal-wrapper" ref={terminalRef} />
                    </div>
                    <div className="v2-input-bar">
                        <div className="v2-input-wrapper">
                            <span className="v2-input-prompt">$</span>
                            <input
                                type="text"
                                value={command}
                                onChange={(e) => setCommand(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleRun();
                                    }
                                }}
                                className="v2-input-field"
                                placeholder="Type a command..."
                            />
                        </div>
                        <button className="v2-run-btn" onClick={handleRun}>
                            Run
                        </button>
                    </div>
                    <ShortcutsBar onSendKey={sendKey} />
                </div>
            </div>

            <div className="command-success-features">
                <h3>Features</h3>
                <ul>
                    <li><strong>Fake Bash Server</strong> - Full shell running in the browser</li>
                    <li><strong>Real Commands</strong> - Try ls, cd, pwd, echo, cat, tree, ps, whoami</li>
                    <li><strong>ANSI Colors</strong> - Type <code>colors</code> to see color support</li>
                    <li><strong>Help</strong> - Type <code>help</code> for available commands</li>
                </ul>
            </div>

            <div className="command-success-commands">
                <h3>Try These Commands</h3>
                <div className="command-success-command-list">
                    <code onClick={() => handleSubmit('ls')}>ls</code>
                    <code onClick={() => handleSubmit('pwd')}>pwd</code>
                    <code onClick={() => handleSubmit('help')}>help</code>
                    <code onClick={() => handleSubmit('colors')}>colors</code>
                    <code onClick={() => handleSubmit('tree')}>tree</code>
                    <code onClick={() => handleSubmit('ps')}>ps</code>
                    <code onClick={() => handleSubmit('whoami')}>whoami</code>
                    <code onClick={() => handleSubmit('date')}>date</code>
                    <code onClick={() => handleSubmit('echo hello')}>echo hello</code>
                    <code onClick={() => handleSubmit('cat readme.md')}>cat readme.md</code>
                </div>
            </div>
        </div>
    );
}

export default XtermQuickTerminal;
