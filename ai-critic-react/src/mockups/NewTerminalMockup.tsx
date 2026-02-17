import { useState, useRef, useEffect } from 'react';

interface NewTerminalMockupProps {
    // No props needed for mockup
}

interface TerminalTab {
    id: string;
    name: string;
    history: LogLine[];
    active: boolean;
    exited: boolean;
    streaming?: boolean;
    disconnected?: boolean;
}

interface LogLine {
    text: string;
    type: 'command' | 'output' | 'error';
    streaming?: boolean;
}

type TerminalVariant = 'normal' | 'streaming' | 'exited' | 'disconnected';

type CommandMock = LogLine[] | ((args: string[]) => LogLine[]);

const COMMAND_MOCKS: Record<string, CommandMock> = {
    'ls': [
        { text: 'total 32', type: 'output' },
        { text: 'drwxr-xr-x  5 user  staff   160 Jan 15 10:30 .', type: 'output' },
        { text: 'drwxr-xr-x  5 user  staff   160 Jan 15 09:00 ..', type: 'output' },
        { text: '-rw-r--r--  1 user  staff  1234 Jan 14 15:45 package.json', type: 'output' },
        { text: '-rw-r--r--  1 user  staff  5678 Jan 13 12:00 src/', type: 'output' },
        { text: '-rw-r--r--  1 user  staff   234 Jan 12 14:30 tsconfig.json', type: 'output' },
        { text: 'drwxr-xr-x  3 user  staff   128 Jan 11 11:00 node_modules/', type: 'output' },
    ],
    'pwd': [
        { text: '/Users/user/project', type: 'output' },
    ],
    'whoami': [
        { text: 'user', type: 'output' },
    ],
    'date': [
        { text: 'Mon Feb 16 10:30:00 UTC 2026', type: 'output' },
    ],
    'echo': (args: string[]) => [
        { text: args.join(' '), type: 'output' },
    ],
    'help': [
        { text: 'Available commands: ls, pwd, whoami, date, echo, help, clear, exit', type: 'output' },
    ],
    'clear': [],
    'exit': [
        { text: 'exit', type: 'command' },
    ],
};

let tabCounter = 1;

export function NewTerminalMockup(_props: NewTerminalMockupProps) {
    const [variant, setVariant] = useState<TerminalVariant>('normal');
    const [showKeyboard, setShowKeyboard] = useState(false);
    const [ctrlMode, setCtrlMode] = useState(false);
    const [tabs, setTabs] = useState<TerminalTab[]>([
        {
            id: 'tab-1',
            name: 'bash',
            history: [
                { text: 'Welcome to mobile terminal mockup', type: 'output' },
                { text: 'Type commands and press Enter to execute', type: 'output' },
                { text: '', type: 'output' },
            ],
            active: true,
            exited: false,
        },
    ]);
    const [activeTabId, setActiveTabId] = useState('tab-1');
    const inputRef = useRef<HTMLDivElement>(null);
    const outputRef = useRef<HTMLDivElement>(null);
    const shortcutsRef = useRef<HTMLDivElement>(null);

    const activeTab = tabs.find(t => t.id === activeTabId) || tabs[0];

    useEffect(() => {
        if (outputRef.current) {
            outputRef.current.scrollTop = outputRef.current.scrollHeight;
        }
    }, [activeTab?.history]);

    useEffect(() => {
        if (showKeyboard && shortcutsRef.current) {
            const keyboardHeight = window.innerHeight - (window.visualViewport?.height || window.innerHeight);
            if (keyboardHeight > 100) {
                shortcutsRef.current.style.bottom = `${keyboardHeight + 10}px`;
            }
        } else if (shortcutsRef.current) {
            shortcutsRef.current.style.bottom = '';
        }
    }, [showKeyboard]);

    useEffect(() => {
        if (variant === 'streaming' && activeTab) {
            const streamingTexts = [
                'Building...',
                'Compiling assets...',
                'Optimizing images...',
                'Generating chunks...',
                'Done!',
            ];
            let idx = 0;
            
            const addNextLine = () => {
                if (idx >= streamingTexts.length || variant !== 'streaming') return;
                
                setTabs(tabs.map(t => 
                    t.id === activeTabId 
                        ? { 
                            ...t, 
                            history: [
                                ...t.history, 
                                { text: streamingTexts[idx], type: 'output', streaming: true }
                            ] 
                        }
                        : t
                ));
                idx++;
                
                if (idx < streamingTexts.length) {
                    setTimeout(addNextLine, 500 + Math.random() * 500);
                }
            };
            
            addNextLine();
        }
    }, [variant]);

    const handleInputKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            executeCommand();
        }
    };

    const executeCommand = () => {
        if (!inputRef.current || !activeTab) return;
        
        const text = inputRef.current.textContent?.trim() || '';

        const parts = text.split(' ');
        const cmd = parts[0].toLowerCase();
        const args = parts.slice(1);

        let newHistory: LogLine[];
        
        if (cmd === 'clear') {
            newHistory = [];
        } else if (cmd === 'exit') {
            newHistory = [
                ...activeTab.history,
                { text: `$ ${text}`, type: 'command' },
                { text: '', type: 'output' },
            ];
            setTabs(tabs.map(t => 
                t.id === activeTabId 
                    ? { ...t, history: newHistory, exited: true }
                    : t
            ));
            if (inputRef.current) inputRef.current.textContent = '';
            return;
        } else if (COMMAND_MOCKS[cmd]) {
            const mock = COMMAND_MOCKS[cmd];
            newHistory = [
                ...activeTab.history,
                { text: `$ ${text}`, type: 'command' },
            ];
            if (typeof mock === 'function') {
                newHistory.push(...mock(args));
            } else {
                newHistory.push(...mock);
            }
            newHistory.push({ text: '', type: 'output' });
        } else if (cmd) {
            newHistory = [
                ...activeTab.history,
                { text: `$ ${text}`, type: 'command' },
                { text: `bash: ${cmd}: command not found`, type: 'error' },
                { text: '', type: 'output' },
            ];
        } else {
            newHistory = [
                ...activeTab.history,
                { text: `$ ${text}`, type: 'command' },
                { text: '', type: 'output' },
            ];
        }
        
        setTabs(tabs.map(t => 
            t.id === activeTabId 
                ? { ...t, history: newHistory }
                : t
        ));
        if (inputRef.current) inputRef.current.textContent = '';
    };

    const addNewTab = () => {
        tabCounter++;
        const newTab: TerminalTab = {
            id: `tab-${tabCounter}`,
            name: 'bash',
            history: [
                { text: `Terminal ${tabCounter}`, type: 'output' },
                { text: '', type: 'output' },
            ],
            active: true,
            exited: false,
        };
        setTabs([...tabs.map(t => ({ ...t, active: false })), newTab]);
        setActiveTabId(newTab.id);
    };

    const handleTabClick = (tabId: string) => {
        setActiveTabId(tabId);
        setTabs(tabs.map(t => ({ ...t, active: t.id === tabId })));
    };

    const handleCloseTab = (tabId: string, e: React.MouseEvent) => {
        e.stopPropagation();
        const newTabs = tabs.filter(t => t.id !== tabId);
        if (newTabs.length === 0) {
            tabCounter++;
            const newTab: TerminalTab = {
                id: `tab-${tabCounter}`,
                name: 'bash',
                history: [
                    { text: `Terminal ${tabCounter}`, type: 'output' },
                    { text: '', type: 'output' },
                ],
                active: true,
                exited: false,
            };
            setTabs([newTab]);
            setActiveTabId(newTab.id);
        } else if (activeTabId === tabId) {
            setActiveTabId(newTabs[newTabs.length - 1].id);
            setTabs(newTabs.map((t, i) => ({ ...t, active: i === newTabs.length - 1 })));
        } else {
            setTabs(newTabs);
        }
    };

    const handleAnyKey = () => {
        if (activeTab?.exited) {
            // Restart the terminal
            setTabs(tabs.map(t => 
                t.id === activeTabId 
                    ? { 
                        ...t, 
                        exited: false, 
                        history: [
                            { text: 'Process terminated', type: 'output' },
                            { text: '', type: 'output' },
                        ]
                    }
                    : t
            ));
        }
    };

    const sendKey = (key: string) => {
        if (!inputRef.current || !activeTab || activeTab.exited) return;
        
        if (ctrlMode) {
            const charCode = key.toLowerCase().charCodeAt(0);
            if (charCode >= 97 && charCode <= 122) {
                inputRef.current.textContent += String.fromCharCode(charCode - 96);
            }
            setCtrlMode(false);
        } else {
            if (key === '\t') {
                inputRef.current.textContent += '    ';
            } else {
                inputRef.current.textContent += key;
            }
        }
        inputRef.current.focus();
    };

    const handleQuickAction = (action: string) => {
        if (!inputRef.current || !activeTab || activeTab.exited) return;

        switch (action) {
            case 'Tab':
                sendKey('\t');
                break;
            case '←':
                sendKey('\x1b[D');
                break;
            case '→':
                sendKey('\x1b[C');
                break;
            case '↑':
                sendKey('\x1b[A');
                break;
            case '↓':
                sendKey('\x1b[B');
                break;
            case 'Esc':
                sendKey('\x1b');
                break;
            case '^C':
                sendKey('\x03');
                break;
            case '^R':
                sendKey('\x12');
                break;
            case 'Paste':
                navigator.clipboard.readText().then(text => {
                    if (inputRef.current) {
                        inputRef.current.textContent += text;
                        inputRef.current.focus();
                    }
                }).catch(() => {});
                break;
        }
    };

    const getLineStyle = (type: LogLine['type'], streaming?: boolean): React.CSSProperties => {
        if (streaming) {
            return { color: '#22d3ee' };
        }
        switch (type) {
            case 'command':
                return { color: '#22c55e' };
            case 'error':
                return { color: '#ef4444' };
            default:
                return { color: '#e2e8f0' };
        }
    };

    return (
        <div style={{ padding: '20px', maxWidth: '500px', margin: '0 auto', background: '#0f172a', minHeight: '100vh', overflowY: 'auto', WebkitOverflowScrolling: 'touch' }}>
            <style>{`
                @viewport {
                    user-zoom: fixed;
                }
                input, textarea, [contenteditable] {
                    font-size: 16px !important;
                }
                @keyframes blink {
                    0%, 50% { opacity: 1; }
                    51%, 100% { opacity: 0; }
                }
            `}</style>
            <div style={{ marginBottom: '16px', paddingBottom: '12px', borderBottom: '1px solid #334155' }}>
                <h3 style={{ margin: 0, fontSize: '16px', color: '#e2e8f0' }}>New Terminal</h3>
            </div>
            
            <div style={{ display: 'flex', gap: '6px', marginBottom: '16px', flexWrap: 'wrap' }}>
                {(['normal', 'streaming', 'exited', 'disconnected'] as TerminalVariant[]).map(v => (
                    <button
                        key={v}
                        onClick={() => {
                            setVariant(v);
                            if (v === 'exited') {
                                setTabs([{
                                    id: 'tab-1',
                                    name: 'bash',
                                    history: [
                                        { text: '$ ./build.sh', type: 'command' },
                                        { text: 'Build completed successfully', type: 'output' },
                                        { text: '', type: 'output' },
                                    ],
                                    active: true,
                                    exited: true,
                                }]);
                            } else if (v === 'disconnected') {
                                setTabs([{
                                    id: 'tab-1',
                                    name: 'bash',
                                    history: [
                                        { text: 'Welcome to mobile terminal mockup', type: 'output' },
                                        { text: 'Type commands and press Enter to execute', type: 'output' },
                                        { text: '', type: 'output' },
                                    ],
                                    active: true,
                                    exited: false,
                                    disconnected: true,
                                }]);
                            } else if (v === 'streaming') {
                                setTabs([{
                                    id: 'tab-1',
                                    name: 'bash',
                                    history: [
                                        { text: '$ npm run build', type: 'command' },
                                        { text: '', type: 'output' },
                                    ],
                                    active: true,
                                    exited: false,
                                }]);
                            } else {
                                setTabs([{
                                    id: 'tab-1',
                                    name: 'bash',
                                    history: [
                                        { text: 'Welcome to mobile terminal mockup', type: 'output' },
                                        { text: 'Type commands and press Enter to execute', type: 'output' },
                                        { text: '', type: 'output' },
                                    ],
                                    active: true,
                                    exited: false,
                                }]);
                            }
                        }}
                        style={{
                            padding: '6px 12px',
                            fontSize: '12px',
                            borderRadius: '4px',
                            border: 'none',
                            cursor: 'pointer',
                            background: variant === v ? '#3b82f6' : '#334155',
                            color: variant === v ? '#fff' : '#94a3b8',
                        }}
                    >
                        {v.charAt(0).toUpperCase() + v.slice(1)}
                    </button>
                ))}
            </div>
                
            <div style={{ background: '#0f172a', borderRadius: '8px', overflow: 'hidden', marginBottom: '20px' }}>
                <div style={{ display: 'flex', gap: '4px', padding: '8px 12px', background: '#1e293b', borderBottom: '1px solid #334155', overflowX: 'auto' }}>
                    {tabs.map(tab => (
                        <span 
                            key={tab.id}
                            onClick={() => handleTabClick(tab.id)}
                            style={{ 
                                padding: '4px 12px', 
                                background: tab.active ? '#0f172a' : '#334155', 
                                borderRadius: '4px', 
                                fontSize: '12px', 
                                color: tab.active ? '#fff' : '#94a3b8',
                                cursor: 'pointer',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '6px',
                                whiteSpace: 'nowrap',
                            }}
                        >
                            {tab.name}
                            {tabs.length > 1 && (
                                <span 
                                    onClick={(e) => handleCloseTab(tab.id, e)}
                                    style={{ fontSize: '14px', color: '#64748b', marginLeft: '2px' }}
                                >×</span>
                            )}
                        </span>
                    ))}
                    <span 
                        onClick={addNewTab}
                        style={{ 
                            padding: '4px 10px', 
                            background: '#334155', 
                            borderRadius: '4px', 
                            fontSize: '12px', 
                            color: '#94a3b8',
                            cursor: 'pointer',
                        }}
                    >
                        +
                    </span>
                </div>
                
                <div 
                    ref={outputRef}
                    onClick={() => inputRef.current?.focus()}
                    style={{ 
                        padding: '12px', 
                        height: '200px', 
                        overflowY: 'auto', 
                        WebkitOverflowScrolling: 'touch',
                        touchAction: 'pan-y',
                        fontFamily: 'monospace', 
                        fontSize: '12px', 
                        lineHeight: '1.5', 
                        textAlign: 'left',
                        cursor: 'text'
                    }}
                >
                    {activeTab.history.map((line, i) => (
                        <div 
                            key={i} 
                            style={{ 
                                ...getLineStyle(line.type, line.streaming), 
                                whiteSpace: 'pre-wrap' 
                            }}
                        >
                            {line.text}
                            {line.streaming && i === activeTab.history.length - 1 && <span style={{ animation: 'blink 1s infinite' }}>▋</span>}
                        </div>
                    ))}
                    
                    {activeTab.exited ? (
                        <div>
                            <div style={{ color: '#f59e0b', marginBottom: '8px' }}>Exit status: exited</div>
                            <div style={{ color: '#64748b', cursor: 'pointer' }} onClick={handleAnyKey}>Press any key to exit...</div>
                        </div>
                    ) : activeTab.disconnected ? (
                        <div style={{ color: '#ef4444', padding: '20px', textAlign: 'center' }}>
                            <div style={{ fontSize: '24px', marginBottom: '8px' }}>⚡</div>
                            <div>Disconnected</div>
                            <button 
                                onClick={() => {
                                    setVariant('normal');
                                    setTabs([{
                                        id: 'tab-1',
                                        name: 'bash',
                                        history: [
                                            { text: 'Reconnecting...', type: 'output' },
                                            { text: 'Connected', type: 'output' },
                                            { text: '', type: 'output' },
                                        ],
                                        active: true,
                                        exited: false,
                                    }]);
                                }}
                                style={{
                                    marginTop: '12px',
                                    padding: '8px 16px',
                                    background: '#3b82f6',
                                    border: 'none',
                                    borderRadius: '4px',
                                    color: '#fff',
                                    cursor: 'pointer',
                                }}
                            >
                                Reconnect
                            </button>
                        </div>
                    ) : (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                            <span style={{ color: '#22c55e' }}>$</span>
                            <div
                                ref={inputRef}
                                contentEditable
                                suppressContentEditableWarning
                                onKeyDown={handleInputKeyDown}
                                onFocus={() => setShowKeyboard(true)}
                                onBlur={() => setTimeout(() => setShowKeyboard(false), 200)}
                                style={{
                                    background: 'transparent',
                                    border: 'none',
                                    color: '#e2e8f0',
                                    fontFamily: 'monospace',
                                    fontSize: '16px',
                                    outline: 'none',
                                    flex: 1,
                                    minWidth: '100px',
                                    caretColor: '#60a5fa',
                                }}
                            />
                        </div>
                    )}
                </div>

                <div 
                    ref={shortcutsRef}
                    style={{ 
                        display: activeTab?.disconnected ? 'none' : 'flex', 
                        gap: '4px', 
                        padding: '8px 12px', 
                        paddingBottom: 'max(8px, env(safe-area-inset-bottom))',
                        background: '#1e293b', 
                        borderTop: '1px solid #334155',
                        overflowX: 'auto',
                        transition: 'bottom 0.2s ease',
                        ...(showKeyboard ? { position: 'fixed', left: 0, right: 0, zIndex: 100 } : {})
                    }}
                >
                    <button style={shortcutBtnStyle} onClick={() => handleQuickAction('Tab')}>Tab</button>
                    <button style={shortcutBtnStyle} onClick={() => handleQuickAction('←')}>←</button>
                    <button style={shortcutBtnStyle} onClick={() => handleQuickAction('→')}>→</button>
                    <button style={shortcutBtnStyle} onClick={() => handleQuickAction('↑')}>↑</button>
                    <button style={shortcutBtnStyle} onClick={() => handleQuickAction('↓')}>↓</button>
                    <button 
                        style={ctrlMode ? { ...shortcutBtnStyle, background: '#60a5fa', color: 'white', borderColor: '#60a5fa' } : shortcutBtnStyle} 
                        onClick={() => setCtrlMode(!ctrlMode)}
                    >Ctrl</button>
                    <button style={shortcutBtnStyle} onClick={() => handleQuickAction('Esc')}>Esc</button>
                    <button style={shortcutBtnStyle} onClick={() => handleQuickAction('^C')}>^C</button>
                    <button style={shortcutBtnStyle} onClick={() => handleQuickAction('^R')}>^R</button>
                    <button style={shortcutBtnStyle} onClick={() => handleQuickAction('Paste')}>Paste</button>
                </div>
            </div>

            <div style={{ background: '#f9fafb', borderRadius: '8px', padding: '16px', marginBottom: '16px', textAlign: 'left' }}>
                <h4 style={{ margin: '0 0 12px 0', fontSize: '14px', color: '#374151' }}>Try these commands:</h4>
                <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '13px', color: '#6b7280' }}>
                    <li style={{ marginBottom: '6px' }}><code style={{ color: '#22c55e' }}>ls</code> - list files</li>
                    <li style={{ marginBottom: '6px' }}><code style={{ color: '#22c55e' }}>pwd</code> - print working directory</li>
                    <li style={{ marginBottom: '6px' }}><code style={{ color: '#22c55e' }}>exit</code> - simulate exit (shows exited status)</li>
                    <li style={{ marginBottom: '6px' }}><code style={{ color: '#22c55e' }}>clear</code> - clear terminal</li>
                    <li style={{ marginBottom: '6px' }}>Click <b>+</b> to add new terminal tab</li>
                </ul>
            </div>

            <p style={{ fontSize: '13px', color: '#92400e', background: '#fef3c7', padding: '12px', borderRadius: '6px', margin: 0 }}>
                Use the buttons above to switch variants: Normal, Streaming, Exited, Disconnected.
            </p>
        </div>
    );
}

const shortcutBtnStyle: React.CSSProperties = {
    padding: '6px 8px',
    background: '#334155',
    border: '1px solid #475569',
    borderRadius: '4px',
    color: '#94a3b8',
    fontSize: '11px',
    fontFamily: 'monospace',
    cursor: 'pointer',
    whiteSpace: 'nowrap',
    flexShrink: 0,
};
