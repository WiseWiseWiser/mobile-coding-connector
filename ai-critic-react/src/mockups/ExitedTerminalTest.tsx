import { useState } from 'react';

interface TerminalTab {
    id: string;
    name: string;
    history: LogLine[];
    exited: boolean;
}

interface LogLine {
    text: string;
    type: 'command' | 'output';
}

let renderCount = 0;

export function ExitedTerminalTest() {
    renderCount++;
    console.log('[RENDER]', renderCount);
    
    const [tabs, setTabs] = useState<TerminalTab[]>([
        {
            id: 'tab-1',
            name: 'bash',
            history: [
                { text: '$ ./build.sh', 'type': 'command' },
                { text: 'Build completed successfully', 'type': 'output' },
                { text: '', type: 'output' },
            ],
            exited: true,
        },
    ]);

    console.log('[STATE] tabs[0].exited:', tabs[0].exited);

    const handleAnyKey = () => {
        console.log('[RESTART] Called');
        setTabs(currentTabs => {
            const tab = currentTabs[0];
            if (tab?.exited) {
                console.log('[RESTART] Restarting...');
                const newTabs: TerminalTab[] = currentTabs.map(t => 
                    t.id === tab.id 
                        ? { 
                            ...t, 
                            exited: false, 
                            history: [
                                ...t.history,
                                { text: 'Process terminated', type: 'output' as const },
                                { text: '', type: 'output' as const },
                            ]
                        }
                        : t
                );
                console.log('[RESTART] newTabs[0].exited:', newTabs[0].exited);
                return newTabs;
            }
            return currentTabs;
        });
    };

    const handleOutputKeyDown = (e: React.KeyboardEvent) => {
        console.log('[KEYDOWN]', e.key);
        if (tabs[0].exited) {
            e.preventDefault();
            handleAnyKey();
        }
    };

    return (
        <div style={{ padding: '20px', maxWidth: '500px', margin: '0 auto', background: '#0f172a', minHeight: '100vh' }}>
            <h3 style={{ margin: '0 0 16px 0', fontSize: '16px', color: '#e2e8f0' }}>Exited Terminal Test</h3>
            <div style={{ color: '#94a3b8', fontSize: '12px', marginBottom: '16px' }}>Render count: {renderCount} | Tabs exited: {tabs[0].exited ? 'YES' : 'NO'}</div>
            
            <div style={{ background: '#0f172a', borderRadius: '8px', overflow: 'hidden', marginBottom: '20px' }}>
                <div style={{ padding: '8px 12px', background: '#1e293b', borderBottom: '1px solid #334155' }}>
                    <span style={{ color: '#94a3b8', fontSize: '12px' }}>exited</span>
                </div>
                
                <div 
                    style={{ 
                        padding: '12px', 
                        height: '200px', 
                        overflowY: 'auto', 
                        fontFamily: 'monospace', 
                        fontSize: '12px', 
                        lineHeight: '1.5',
                    }}
                    onKeyDown={handleOutputKeyDown}
                    tabIndex={tabs[0].exited ? 0 : undefined}
                >
                    {tabs[0].history.map((line, i) => (
                        <div 
                            key={i} 
                            style={{ 
                                color: line.type === 'command' ? '#22c55e' : '#e2e8f0', 
                                whiteSpace: 'pre-wrap' 
                            }}
                        >
                            {line.text}
                        </div>
                    ))}
                    
                    {tabs[0].exited && (
                        <div>
                            <div style={{ color: '#f59e0b', marginBottom: '8px' }}>Exit status: exited</div>
                            <div 
                                style={{ color: '#64748b', padding: '8px', background: '#1e293b', borderRadius: '4px' }} 
                            >
                                Press any key to restart...
                            </div>
                        </div>
                    )}
                </div>
            </div>

            <div style={{ background: '#f9fafb', borderRadius: '8px', padding: '16px', color: '#6b7280', fontSize: '13px' }}>
                <p style={{ margin: '0 0 8px 0' }}>This page has only an exited terminal.</p>
                <p style={{ margin: 0 }}>Type any key to restart (clicking should do nothing).</p>
            </div>
        </div>
    );
}
