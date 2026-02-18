import { useState } from 'react';
import { CommandInput, type DropdownPosition } from '../pure-view/CommandInput';
import './CommandInputMockup.css';

const defaultHistory = [
    'npm install',
    'npm run dev',
    'npm test',
    'npm run build',
    'git status',
    'git add .',
    'git commit -m "fix"',
    'git push origin main',
    'ls -la',
    'cat package.json',
    'echo "hello world"',
    'cd /home/user',
    'mkdir newfolder',
    'rm -rf node_modules',
    'pwd',
];

export function CommandInputMockup() {
    const [command, setCommand] = useState('');
    const [commandTop, setCommandTop] = useState('');
    const [history, setHistory] = useState<string[]>(defaultHistory);
    const [output, setOutput] = useState<string[]>([]);
    const [dropdownPosition, setDropdownPosition] = useState<DropdownPosition>('bottom');

    const handleSubmit = (cmd: string) => {
        const trimmed = cmd.trim();
        if (!trimmed) return;

        setHistory(prev => {
            const filtered = prev.filter(h => h !== trimmed);
            return [trimmed, ...filtered].slice(0, 20);
        });

        setOutput(prev => [...prev, `$ ${trimmed}`, `> Executed: ${trimmed}`]);
        setCommand('');
        setCommandTop('');
    };

    return (
        <div className="command-input-mockup">
            <div className="command-input-mockup-header">
                <h2>Command Input</h2>
                <p>
                    A command input with history dropdown, fuzzy search, and keyboard navigation.
                    Try typing to see the dropdown, use arrow keys to navigate, Tab to autocomplete.
                </p>
            </div>

            <div className="command-input-mockup-section">
                <div className="command-input-position-selector">
                    <label>Dropdown Position:</label>
                    <select 
                        value={dropdownPosition} 
                        onChange={(e) => setDropdownPosition(e.target.value as DropdownPosition)}
                    >
                        <option value="bottom">Bottom</option>
                        <option value="top">Top</option>
                    </select>
                </div>
            </div>

            <div className="command-input-mockup-section">
                <h3>Interactive Demo (position: {dropdownPosition})</h3>
                <div className="command-input-demo">
                    <CommandInput
                        value={command}
                        onChange={setCommand}
                        onSubmit={handleSubmit}
                        history={history}
                        placeholder="Type a command... (try 'npm' or 'git')"
                        dropdownPosition={dropdownPosition}
                    />
                    <button 
                        className="command-input-run-btn"
                        onClick={() => handleSubmit(command)}
                    >
                        Run
                    </button>
                </div>
            </div>

            <div className="command-input-mockup-section">
                <h3>Both Positions Demo</h3>
                <div className="command-input-variants">
                    <div className="command-input-variant">
                        <span className="variant-label">Top</span>
                        <div className="command-input-demo compact">
                            <CommandInput
                                value={commandTop}
                                onChange={setCommandTop}
                                onSubmit={handleSubmit}
                                history={history}
                                placeholder="Dropdown above"
                                dropdownPosition="top"
                            />
                        </div>
                    </div>
                    <div className="command-input-variant">
                        <span className="variant-label">Bottom (default)</span>
                        <div className="command-input-demo compact">
                            <CommandInput
                                value={command}
                                onChange={setCommand}
                                onSubmit={handleSubmit}
                                history={history}
                                placeholder="Dropdown below"
                                dropdownPosition="bottom"
                            />
                        </div>
                    </div>
                </div>
            </div>

            <div className="command-input-mockup-section">
                <h3>Features</h3>
                <ul className="command-input-features">
                    <li><strong>History dropdown</strong> - Shows recent commands as you type</li>
                    <li><strong>Fuzzy search</strong> - Filters commands that match anywhere in the string</li>
                    <li><strong>Keyboard navigation</strong> - Arrow keys, Tab, Enter, Escape</li>
                    <li><strong>iOS compatible</strong> - 16px font prevents zoom on focus</li>
                    <li><strong>Autocomplete</strong> - Tab cycles through matching commands</li>
                    <li><strong>Position option</strong> - Dropdown can appear above or below input</li>
                </ul>
            </div>

            <div className="command-input-mockup-section">
                <h3>Output Log</h3>
                <div className="command-input-output">
                    {output.map((line, i) => (
                        <div 
                            key={i} 
                            className={`command-input-output-line ${line.startsWith('$') ? 'command' : 'result'}`}
                        >
                            {line}
                        </div>
                    ))}
                    {output.length === 0 && (
                        <div className="command-input-output-empty">
                            Run a command to see output here...
                        </div>
                    )}
                </div>
            </div>

            <div className="command-input-mockup-section">
                <h3>Keyboard Shortcuts</h3>
                <div className="command-input-shortcuts">
                    <div className="command-input-shortcut">
                        <kbd>↑</kbd> / <kbd>↓</kbd>
                        <span>Navigate history</span>
                    </div>
                    <div className="command-input-shortcut">
                        <kbd>Enter</kbd>
                        <span>Execute command</span>
                    </div>
                    <div className="command-input-shortcut">
                        <kbd>Tab</kbd>
                        <span>Cycle autocomplete</span>
                    </div>
                    <div className="command-input-shortcut">
                        <kbd>Esc</kbd>
                        <span>Close dropdown</span>
                    </div>
                </div>
            </div>
        </div>
    );
}

export default CommandInputMockup;
