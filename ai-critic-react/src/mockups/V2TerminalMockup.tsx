import { useState, useRef, useEffect } from 'react';
import '@xterm/xterm/css/xterm.css';
import { useFakeTerminal } from './fake-server';
import { XtermQuickTerminal } from './CommandSuccessTerminal';
import './V2TerminalMockup.css';

interface TerminalCaseProps {
    title: string;
    description: string;
}

// Case 1: Connecting State
function ConnectingTerminal({ title, description }: TerminalCaseProps) {
    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className="v2-terminal-status connecting">
                        <span className="v2-status-dot"></span>
                        Connecting
                    </div>
                    <span className="v2-terminal-title">bash</span>
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content">
                        <div className="v2-connecting-state">
                            <div className="v2-spinner"></div>
                            <span>Connecting to server...</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}

// Case 2: Connected Ready State (Interactive) - Using fake shell
function ConnectedReadyTerminal({ title, description }: TerminalCaseProps) {
    const containerRef = useRef<HTMLDivElement>(null);

    const {
        terminalRef,
        connected,
        sendKey,
        reconnect,
    } = useFakeTerminal({
        cwd: '/home/user',
        name: 'mock-shell',
    });

    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className={`v2-terminal-status ${connected ? 'connected' : 'disconnected'}`}>
                        <span className="v2-status-dot"></span>
                        {connected ? 'Connected' : 'Disconnected'}
                    </div>
                    <span className="v2-terminal-title">bash</span>
                    {!connected && (
                        <button className="v2-reconnect-btn-small" onClick={reconnect}>
                            Reconnect
                        </button>
                    )}
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content v2-fake-terminal-wrapper" ref={containerRef}>
                        <div className="v2-terminal-container-inner" ref={terminalRef} />
                    </div>
                </div>
                <div className="v2-shortcuts-bar">
                    <button className="v2-shortcut-btn" onClick={() => sendKey('\t')}>Tab</button>
                    <button className="v2-shortcut-btn" onClick={() => sendKey('\x1b[A')}>↑</button>
                    <button className="v2-shortcut-btn" onClick={() => sendKey('\x1b[B')}>↓</button>
                    <button className="v2-shortcut-btn" onClick={() => sendKey('\x1b')}>Esc</button>
                    <button className="v2-shortcut-btn" onClick={() => sendKey('\x03')}>^C</button>
                </div>
            </div>
        </div>
    );
}

// Case 3: With Command History (Static)
function WithHistoryTerminal({ title, description }: TerminalCaseProps) {
    const history = [
        { text: '$ ls -la', type: 'command' },
        { text: 'drwxr-xr-x  5 user  staff   160 Jan 15 10:30 .', type: 'output' },
        { text: 'drwxr-xr-x  5 user  staff   160 Jan 15 09:00 ..', type: 'output' },
        { text: '-rw-r--r--  1 user  staff  1234 Jan 14 15:45 package.json', type: 'output' },
        { text: '', type: 'output' },
        { text: '$ git status', type: 'command' },
        { text: 'On branch main', type: 'output' },
        { text: 'Your branch is up to date with "origin/main".', type: 'output' },
        { text: '', type: 'output' },
        { text: 'nothing to commit, working tree clean', type: 'output' },
    ];

    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className="v2-terminal-status connected">
                        <span className="v2-status-dot"></span>
                        Connected
                    </div>
                    <span className="v2-terminal-title">bash</span>
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content">
                        {history.map((line, i) => (
                            <div 
                                key={i} 
                                className={`v2-terminal-line ${line.type === 'command' ? 'command' : ''}`}
                            >
                                {line.text}
                            </div>
                        ))}
                        <div className="v2-terminal-input-line">
                            <span className="v2-prompt">$</span>
                            <span className="v2-cursor">▋</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}

// Case 4: Disconnected State
function DisconnectedTerminal({ title, description }: TerminalCaseProps) {
    const [isReconnecting, setIsReconnecting] = useState(false);

    const handleReconnect = () => {
        setIsReconnecting(true);
        setTimeout(() => setIsReconnecting(false), 2000);
    };

    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className="v2-terminal-status disconnected">
                        <span className="v2-status-dot"></span>
                        Disconnected
                    </div>
                    <span className="v2-terminal-title">bash</span>
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content v2-disconnected-content">
                        <div className="v2-disconnected-icon">⚡</div>
                        <div className="v2-disconnected-title">Connection Lost</div>
                        <div className="v2-disconnected-message">
                            The connection to the server was interrupted.
                        </div>
                        <button 
                            className="v2-reconnect-btn"
                            onClick={handleReconnect}
                            disabled={isReconnecting}
                        >
                            {isReconnecting ? (
                                <>
                                    <span className="v2-spinner-small"></span>
                                    Reconnecting...
                                </>
                            ) : (
                                'Reconnect'
                            )}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}

// Case 5: Reconnecting State
function ReconnectingTerminal({ title, description }: TerminalCaseProps) {
    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className="v2-terminal-status reconnecting">
                        <span className="v2-status-dot"></span>
                        Reconnecting
                    </div>
                    <span className="v2-terminal-title">bash</span>
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content v2-reconnecting-content">
                        <div className="v2-spinner"></div>
                        <div className="v2-reconnecting-title">Restoring session...</div>
                        <div className="v2-reconnecting-progress">
                            <div className="v2-progress-bar">
                                <div className="v2-progress-fill"></div>
                            </div>
                        </div>
                        <div className="v2-reconnecting-message">
                            Attempting to restore your terminal session
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}

// Case 6: Command Success (Interactive) - Now a standalone component
function CommandSuccessTerminalWrapper({ title, description }: TerminalCaseProps) {
    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <XtermQuickTerminal />
        </div>
    );
}

// Case 7: Command Error (Static)
function CommandErrorTerminal({ title, description }: TerminalCaseProps) {
    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className="v2-terminal-status connected">
                        <span className="v2-status-dot"></span>
                        Connected
                    </div>
                    <span className="v2-terminal-title">bash</span>
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content">
                        <div className="v2-terminal-line command">
                            <span className="v2-prompt">$</span> ./deploy.sh
                        </div>
                        <div className="v2-terminal-line error">
                            Error: Permission denied
                        </div>
                        <div className="v2-terminal-line error">
                            at /usr/local/bin/deploy.sh:15:8
                        </div>
                        <div className="v2-terminal-line error">
                            at processTicksAndRejections (internal/process/task_queues.js:97:5)
                        </div>
                        <div className="v2-error-indicator">
                            <span className="v2-error-icon">✗</span>
                            <span>Exit code: 1</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}

// Case 8: Long Running Process (Interactive)
function LongRunningProcessTerminal({ title, description }: TerminalCaseProps) {
    const [lines, setLines] = useState<string[]>([
        '> docker build -t myapp:latest .',
        'Sending build context to Docker daemon  15.3MB',
    ]);
    const [isRunning, setIsRunning] = useState(true);
    const [isPaused, setIsPaused] = useState(false);

    useEffect(() => {
        if (!isRunning || isPaused) return;

        const messages = [
            'Step 1/8 : FROM node:18-alpine',
            ' ---> 8d3f1437b0f3',
            'Step 2/8 : WORKDIR /app',
            ' ---> Running in 4a2b3c1d5e6f',
            ' ---> 9e8f7g6h5i4j',
            'Step 3/8 : COPY package*.json ./',
            ' ---> 1a2b3c4d5e6f',
            'Step 4/8 : RUN npm ci --only=production',
            ' ---> Running in 7g8h9i0j1k2l',
        ];

        let index = 0;
        const interval = setInterval(() => {
            if (index < messages.length) {
                setLines(prev => [...prev, messages[index]]);
                index++;
            } else {
                setIsRunning(false);
                setLines(prev => [...prev, 'Successfully built myapp:latest']);
            }
        }, 1500);

        return () => clearInterval(interval);
    }, [isRunning, isPaused]);

    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className="v2-terminal-status connected">
                        <span className="v2-status-dot"></span>
                        Connected
                    </div>
                    <span className="v2-terminal-title">bash</span>
                    {isRunning && (
                        <button 
                            className="v2-pause-btn"
                            onClick={() => setIsPaused(!isPaused)}
                        >
                            {isPaused ? 'Resume' : 'Pause'}
                        </button>
                    )}
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content v2-streaming">
                        {lines.map((line, i) => (
                            <div key={i} className="v2-terminal-line">
                                {line}
                                {i === lines.length - 1 && isRunning && (
                                    <span className="v2-streaming-cursor">▋</span>
                                )}
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        </div>
    );
}

// Case 9: Process Exited (Interactive)
function ProcessExitedTerminal({ title, description }: TerminalCaseProps) {
    const [isExited, setIsExited] = useState(true);
    const [showMessage, setShowMessage] = useState(false);
    const inputRef = useRef<HTMLInputElement>(null);

    const handleRestart = () => {
        setIsExited(false);
        setShowMessage(true);
        setTimeout(() => setShowMessage(false), 2000);
    };

    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className={`v2-terminal-status ${isExited ? 'exited' : 'connected'}`}>
                        <span className="v2-status-dot"></span>
                        {isExited ? 'Exited' : 'Connected'}
                    </div>
                    <span className="v2-terminal-title">bash</span>
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content">
                        <div className="v2-terminal-line command">
                            <span className="v2-prompt">$</span> ./build.sh
                        </div>
                        <div className="v2-terminal-line output">
                            Build completed successfully
                        </div>
                        
                        {isExited ? (
                            <div className="v2-exited-state">
                                <div className="v2-exit-status">
                                    Exit status: 0
                                </div>
                                <input
                                    ref={inputRef}
                                    type="text"
                                    className="v2-restart-input"
                                    placeholder="Press any key to restart..."
                                    onKeyDown={handleRestart}
                                    autoComplete="off"
                                />
                            </div>
                        ) : (
                            <>
                                {showMessage && (
                                    <div className="v2-restart-message">
                                        Process terminated
                                    </div>
                                )}
                                <div className="v2-terminal-input-line">
                                    <span className="v2-prompt">$</span>
                                    <span className="v2-cursor">▋</span>
                                </div>
                            </>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}

// Case 10: Keyboard Open (Mobile Demo)
function KeyboardOpenTerminal({ title, description }: TerminalCaseProps) {
    const [input, setInput] = useState('');
    const [isFocused, setIsFocused] = useState(false);

    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className="v2-terminal-status connected">
                        <span className="v2-status-dot"></span>
                        Connected
                    </div>
                    <span className="v2-terminal-title">bash</span>
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content">
                        <div className="v2-terminal-line">Welcome to mobile terminal</div>
                        <div className="v2-terminal-line">Tap below to open keyboard</div>
                        <div className="v2-terminal-input-line">
                            <span className="v2-prompt">$</span>
                            <input
                                type="text"
                                value={input}
                                onChange={(e) => setInput(e.target.value)}
                                onFocus={() => setIsFocused(true)}
                                onBlur={() => setIsFocused(false)}
                                className="v2-terminal-input"
                                placeholder="Type here..."
                                autoComplete="off"
                            />
                            <span className="v2-cursor">▋</span>
                        </div>
                    </div>
                </div>
                <div className={`v2-shortcuts-bar ${isFocused ? 'floating' : ''}`}>
                    <button className="v2-shortcut-btn">Tab</button>
                    <button className="v2-shortcut-btn">←</button>
                    <button className="v2-shortcut-btn">→</button>
                    <button className="v2-shortcut-btn">↑</button>
                    <button className="v2-shortcut-btn">↓</button>
                    <button className="v2-shortcut-btn">Ctrl</button>
                    <button className="v2-shortcut-btn">Paste</button>
                </div>
                {isFocused && (
                    <div className="v2-keyboard-indicator">
                        Keyboard is open • Shortcuts floating above
                    </div>
                )}
            </div>
        </div>
    );
}

// Case 11: No Tabs Empty State
function EmptyStateTerminal({ title, description }: TerminalCaseProps) {
    const [hasTerminal, setHasTerminal] = useState(false);

    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                {!hasTerminal ? (
                    <div className="v2-empty-state">
                        <div className="v2-empty-icon">💻</div>
                        <h3>No Terminal Sessions</h3>
                        <p>Get started by creating a new terminal session</p>
                        <button 
                            className="v2-create-btn"
                            onClick={() => setHasTerminal(true)}
                        >
                            + Create New Terminal
                        </button>
                        <div className="v2-empty-help">
                            <span>💡 Tip: You can have multiple terminals running simultaneously</span>
                        </div>
                    </div>
                ) : (
                    <>
                        <div className="v2-terminal-header">
                            <div className="v2-terminal-status connected">
                                <span className="v2-status-dot"></span>
                                Connected
                            </div>
                            <span className="v2-terminal-title">bash</span>
                        </div>
                        <div className="v2-terminal-body">
                            <div className="v2-terminal-content">
                                <div className="v2-terminal-line">Terminal created successfully!</div>
                                <div className="v2-terminal-input-line">
                                    <span className="v2-prompt">$</span>
                                    <span className="v2-cursor">▋</span>
                                </div>
                            </div>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}

// Case 12: Multiple Tabs
function MultipleTabsTerminal({ title, description }: TerminalCaseProps) {
    const [activeTab, setActiveTab] = useState('bash');
    const [hasNewOutput, setHasNewOutput] = useState<{[key: string]: boolean}>({
        'python': true,
    });

    const tabs = [
        { id: 'bash', name: 'bash', content: ['$ echo "Tab 1"', 'Tab 1'] },
        { id: 'python', name: 'python', content: ['>>> print("Tab 2")', 'Tab 2'] },
        { id: 'node', name: 'node', content: ['> console.log("Tab 3")', 'Tab 3'] },
    ];

    const handleTabClick = (tabId: string) => {
        setActiveTab(tabId);
        setHasNewOutput(prev => ({ ...prev, [tabId]: false }));
    };

    return (
        <div className="v2-terminal-case">
            <div className="v2-terminal-case-header">
                <h4>{title}</h4>
                <span className="v2-case-description">{description}</span>
            </div>
            <div className="v2-terminal-container">
                <div className="v2-terminal-header">
                    <div className="v2-terminal-tabs">
                        {tabs.map(tab => (
                            <button
                                key={tab.id}
                                className={`v2-tab ${activeTab === tab.id ? 'active' : ''} ${hasNewOutput[tab.id] ? 'has-output' : ''}`}
                                onClick={() => handleTabClick(tab.id)}
                            >
                                {tab.name}
                                {hasNewOutput[tab.id] && <span className="v2-new-output-dot"></span>}
                            </button>
                        ))}
                        <button className="v2-tab-add">+</button>
                    </div>
                </div>
                <div className="v2-terminal-body">
                    <div className="v2-terminal-content">
                        {tabs.find(t => t.id === activeTab)?.content.map((line, i) => (
                            <div key={i} className="v2-terminal-line">{line}</div>
                        ))}
                        <div className="v2-terminal-input-line">
                            <span className="v2-prompt">
                                {activeTab === 'bash' ? '$' : activeTab === 'python' ? '>>>' : '>'}
                            </span>
                            <span className="v2-cursor">▋</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}

// Main Component
export function V2TerminalMockup() {
    const cases = [
        { component: ConnectingTerminal, title: '1. Connecting', description: 'Initial connection state with spinner' },
        { component: ConnectedReadyTerminal, title: '2. Connected Ready', description: 'Active terminal ready for input (interactive)' },
        { component: WithHistoryTerminal, title: '3. With History', description: 'Terminal with pre-populated command history' },
        { component: DisconnectedTerminal, title: '4. Disconnected', description: 'Connection lost with reconnect option' },
        { component: ReconnectingTerminal, title: '5. Reconnecting', description: 'Restoring session with progress indicator' },
        { component: CommandSuccessTerminalWrapper, title: '6. Command Success', description: 'Successful command execution with output (interactive)' },
        { component: CommandErrorTerminal, title: '7. Command Error', description: 'Command failed with error message' },
        { component: LongRunningProcessTerminal, title: '8. Long Running Process', description: 'Streaming output with pause/resume (interactive)' },
        { component: ProcessExitedTerminal, title: '9. Process Exited', description: 'Process ended with restart prompt (interactive)' },
        { component: KeyboardOpenTerminal, title: '10. Keyboard Open', description: 'Mobile keyboard handling demonstration' },
        { component: EmptyStateTerminal, title: '11. Empty State', description: 'No terminal sessions with CTA (interactive)' },
        { component: MultipleTabsTerminal, title: '12. Multiple Tabs', description: 'Tab switching with new output indicators' },
    ];

    return (
        <div className="v2-terminal-mockup">
            <div className="v2-mockup-header">
                <h2>V2 Terminal Mockup</h2>
                <p>12 terminal states demonstrating different UX patterns for mobile terminals</p>
            </div>
            <div className="v2-cases-container">
                {cases.map((Case, index) => (
                    <Case.component 
                        key={index}
                        title={Case.title}
                        description={Case.description}
                    />
                ))}
            </div>
        </div>
    );
}
