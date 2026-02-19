// FakeShellServer.ts - A pure frontend shell server that simulates terminal responses

const ANSI = {
    reset: '\x1b[0m',
    bright: '\x1b[1m',
    dim: '\x1b[2m',
    red: '\x1b[31m',
    green: '\x1b[32m',
    yellow: '\x1b[33m',
    blue: '\x1b[34m',
    magenta: '\x1b[35m',
    cyan: '\x1b[36m',
    white: '\x1b[37m',
    brightRed: '\x1b[91m',
    brightGreen: '\x1b[92m',
    brightYellow: '\x1b[93m',
    brightBlue: '\x1b[94m',
    brightMagenta: '\x1b[95m',
    brightCyan: '\x1b[96m',
    brightWhite: '\x1b[97m',
    bgBlue: '\x1b[44m',
    enterAltScreen: '\x1b[?1049h',
    exitAltScreen: '\x1b[?1049l',
    clearScreen: '\x1b[2J',
    cursorHome: '\x1b[H',
};

function escapeAnsiForRegex(str: string): string {
    return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

let lineIdCounter = 0;

interface OutputLine {
    id: number;
    content: string;
}

interface AltScreenState {
    lines: OutputLine[];
    currentLine: string;
    vimState?: { filename: string; keyLog: string[] };
}

interface SessionInternal {
    cwd: string;
    cols: number;
    visibleRows: number;
    inputHistory: string[];
    outputLines: OutputLine[];
    currentLine: string;
    inAltScreen: boolean;
    mainBuffer: { lines: OutputLine[]; currentLine: string };
    altScreenBuffer: AltScreenState | null;
}

export interface FakeShellSession {
    id: string;
    cwd: string;
    cols: number;
    visibleRows: number;
    send: (data: string) => void;
    onData: (callback: (data: string) => void) => () => void;
    onClose: (callback: () => void) => () => void;
    resize: (cols: number, visibleRows: number) => void;
    close: () => void;
    getOutputLines: () => OutputLine[];
    isInAltScreen: () => boolean;
}

export class FakeShellServer {
    private sessions: Map<string, SessionInternal> = new Map();
    private sessionCounter = 0;
    private mockFileSystem: Map<string, string[]> = new Map();
    private dataCallbacks: Map<string, ((data: string) => void)[]> = new Map();
    private closeCallbacks: Map<string, (() => void)[]> = new Map();

    constructor() {
        this.mockFileSystem.set('/', ['bin', 'etc', 'home', 'usr', 'var', 'tmp', '.bashrc', '.profile']);
        this.mockFileSystem.set('/home', ['user']);
        this.mockFileSystem.set('/home/user', [
            'Documents', 'Downloads', 'Desktop', 'Pictures', 'Videos',
            '.bashrc', '.profile', '.ssh', 'projects', 'README.md'
        ]);
        this.mockFileSystem.set('/home/user/projects', ['ai-critic', 'my-app', 'dotfiles']);
        this.mockFileSystem.set('/home/user/Documents', ['notes.txt', 'resume.pdf', 'project-ideas.md']);
    }

    createSession(options?: { cwd?: string; name?: string }): FakeShellSession {
        const initialCwd = options?.cwd ?? '/home/user';
        const sessionId = `fake-session-${++this.sessionCounter}`;

        const internal: SessionInternal = {
            cwd: initialCwd,
            cols: 80,
            visibleRows: 24,
            inputHistory: [],
            outputLines: [],
            currentLine: '',
            inAltScreen: false,
            mainBuffer: { lines: [], currentLine: '' },
            altScreenBuffer: null,
        };

        this.sessions.set(sessionId, internal);
        this.dataCallbacks.set(sessionId, []);
        this.closeCallbacks.set(sessionId, []);

        const session: FakeShellSession = {
            id: sessionId,
            get cwd() { return internal.cwd; },
            get cols() { return internal.cols; },
            get visibleRows() { return internal.visibleRows; },

            send: (data: string) => {
                this.handleInput(sessionId, data);
            },

            onData: (callback: (data: string) => void) => {
                this.dataCallbacks.get(sessionId)?.push(callback);
                return () => {
                    const callbacks = this.dataCallbacks.get(sessionId);
                    if (callbacks) {
                        const index = callbacks.indexOf(callback);
                        if (index > -1) callbacks.splice(index, 1);
                    }
                };
            },

            onClose: (callback: () => void) => {
                this.closeCallbacks.get(sessionId)?.push(callback);
                return () => {
                    const callbacks = this.closeCallbacks.get(sessionId);
                    if (callbacks) {
                        const index = callbacks.indexOf(callback);
                        if (index > -1) callbacks.splice(index, 1);
                    }
                };
            },

            resize: (cols: number, visibleRows: number) => {
                internal.cols = cols;
                internal.visibleRows = visibleRows;
                if (internal.inAltScreen && internal.altScreenBuffer?.vimState) {
                    this.renderVimScreen(sessionId);
                }
            },

            close: () => {
                this.closeCallbacks.get(sessionId)?.forEach(cb => cb());
                this.sessions.delete(sessionId);
                this.dataCallbacks.delete(sessionId);
                this.closeCallbacks.delete(sessionId);
            },

            getOutputLines: () => {
                return [...internal.outputLines];
            },

            isInAltScreen: () => {
                return internal.inAltScreen;
            },
        };

        setTimeout(() => {
            this.showWelcome(sessionId);
            this.showPrompt(sessionId);
        }, 100);

        return session;
    }

    private sendOutput(sessionId: string, data: string) {
        const callbacks = this.dataCallbacks.get(sessionId);
        if (!callbacks) return;

        let processedData = data;

        if (processedData.includes(ANSI.enterAltScreen)) {
            const internal = this.sessions.get(sessionId);
            if (internal) {
                internal.mainBuffer = {
                    lines: [...internal.outputLines],
                    currentLine: internal.currentLine,
                };
                if (!internal.altScreenBuffer) {
                    internal.altScreenBuffer = { lines: [], currentLine: '' };
                } else {
                    internal.altScreenBuffer.lines = [];
                    internal.altScreenBuffer.currentLine = '';
                }
                internal.outputLines = [];
                internal.currentLine = '';
                internal.inAltScreen = true;
            }
            processedData = processedData.replace(new RegExp(escapeAnsiForRegex(ANSI.enterAltScreen), 'g'), '');
        }

        if (processedData.includes(ANSI.exitAltScreen)) {
            const internal = this.sessions.get(sessionId);
            if (internal && internal.altScreenBuffer) {
                internal.outputLines = internal.mainBuffer.lines;
                internal.currentLine = internal.mainBuffer.currentLine;
                internal.altScreenBuffer = null;
                internal.inAltScreen = false;
            }
            processedData = processedData.replace(new RegExp(escapeAnsiForRegex(ANSI.exitAltScreen), 'g'), '');
        }

        if (processedData.includes(ANSI.clearScreen)) {
            const internal = this.sessions.get(sessionId);
            if (internal) {
                internal.outputLines = [];
                internal.currentLine = '';
            }
            processedData = processedData.replace(new RegExp(escapeAnsiForRegex(ANSI.clearScreen), 'g'), '');
        }

        if (processedData.includes(ANSI.cursorHome)) {
            processedData = processedData.replace(new RegExp(escapeAnsiForRegex(ANSI.cursorHome), 'g'), '');
        }

        const internal = this.sessions.get(sessionId);
        if (!internal) {
            callbacks.forEach(cb => cb(''));
            return;
        }

        if (processedData.length === 0) {
            callbacks.forEach(cb => cb(''));
            return;
        }

        if (processedData.includes('\b')) {
            internal.currentLine = this.applyBackspace(internal.currentLine, processedData);
            if (internal.outputLines.length > 0) {
                internal.outputLines[internal.outputLines.length - 1].content = internal.currentLine;
            }
            callbacks.forEach(cb => cb(processedData));
            return;
        }

        if (processedData.includes('\r\n')) {
            const parts = processedData.split('\r\n');
            const firstPart = internal.currentLine + parts[0];

            if (internal.outputLines.length > 0 && internal.currentLine.length > 0) {
                internal.outputLines[internal.outputLines.length - 1].content = firstPart;
            } else if (firstPart.length > 0) {
                internal.outputLines.push({ id: ++lineIdCounter, content: firstPart });
            }

            internal.currentLine = '';

            for (let i = 1; i < parts.length - 1; i++) {
                if (parts[i].length > 0) {
                    internal.outputLines.push({ id: ++lineIdCounter, content: parts[i] });
                }
            }

            const lastPart = parts[parts.length - 1];
            if (lastPart.length > 0) {
                internal.currentLine = lastPart;
                internal.outputLines.push({ id: ++lineIdCounter, content: lastPart });
            }
        } else if (processedData.length > 0) {
            const wasEmpty = internal.currentLine.length === 0;
            internal.currentLine += processedData;

            if (wasEmpty) {
                internal.outputLines.push({ id: ++lineIdCounter, content: internal.currentLine });
            } else if (internal.outputLines.length > 0) {
                internal.outputLines[internal.outputLines.length - 1].content = internal.currentLine;
            } else {
                internal.outputLines.push({ id: ++lineIdCounter, content: internal.currentLine });
            }
        }

        if (internal.outputLines.length > 256) {
            internal.outputLines = internal.outputLines.slice(-256);
        }

        callbacks.forEach(cb => cb(processedData));
    }

    private applyBackspace(line: string, backspaceSeq: string): string {
        let result = line;
        let i = 0;
        while (i < backspaceSeq.length) {
            const char = backspaceSeq[i];
            if (char === '\b' || char === '\x7f') {
                if (i + 2 < backspaceSeq.length && backspaceSeq[i + 1] === ' ' && backspaceSeq[i + 2] === '\b') {
                    result = result.slice(0, -1);
                    i += 3;
                } else {
                    result = result.slice(0, -1);
                    i++;
                }
            } else {
                i++;
            }
        }
        return result;
    }

    private handleInput(sessionId: string, data: string) {
        const internal = this.sessions.get(sessionId);
        if (!internal) return;

        if (internal.inAltScreen && internal.altScreenBuffer?.vimState) {
            this.handleVimInput(sessionId, data);
            return;
        }

        for (let i = 0; i < data.length; i++) {
            const char = data[i];

            if (char === '\r' || char === '\n') {
                this.sendOutput(sessionId, '\r\n');
                const command = internal.inputHistory.join('').trim();
                internal.inputHistory = [];

                if (command) {
                    this.executeCommand(sessionId, command);
                }

                this.showPrompt(sessionId);
            } else if (char === '\x7f' || char === '\b') {
                if (internal.inputHistory.length > 0) {
                    internal.inputHistory.pop();
                    this.sendOutput(sessionId, '\b \b');
                }
            } else if (char === '\x03') {
                internal.inputHistory = [];
                this.sendOutput(sessionId, '^C');
                this.sendOutput(sessionId, '\r\n');
                this.showPrompt(sessionId);
            } else if (char === '\x0c') {
                this.sendOutput(sessionId, ANSI.clearScreen + ANSI.cursorHome);
                this.showPrompt(sessionId);
            } else if (char === '\t') {
                internal.inputHistory.push('    ');
                this.sendOutput(sessionId, '    ');
            } else {
                internal.inputHistory.push(char);
                this.sendOutput(sessionId, char);
            }
        }
    }

    private showWelcome(sessionId: string) {
        const internal = this.sessions.get(sessionId);
        if (!internal) return;

        const width = Math.min(internal.cols, 100);
        const innerWidth = width - 2;

        const topBorder = '═'.repeat(innerWidth);
        const emptyLine = ' '.repeat(innerWidth);

        const title = '🚀 Welcome to Pure Terminal Mockup';
        const titlePadding = Math.max(0, Math.floor((innerWidth - title.length) / 2));
        const titleLine = ' '.repeat(titlePadding) + title + ' '.repeat(innerWidth - titlePadding - title.length);

        const desc = 'This is a fake shell running entirely in the browser.';
        const descPadding = Math.max(0, Math.floor((innerWidth - desc.length) / 2));
        const descLine = ' '.repeat(descPadding) + desc + ' '.repeat(innerWidth - descPadding - desc.length);

        const cmds = 'Try commands like: ls, pwd, echo, help';
        const cmdsPadding = Math.max(0, Math.floor((innerWidth - cmds.length) / 2));
        const cmdsLine = ' '.repeat(cmdsPadding) + cmds + ' '.repeat(innerWidth - cmdsPadding - cmds.length);

        const welcome = [
            '',
            `${ANSI.brightCyan}╔${topBorder}╗${ANSI.reset}`,
            `${ANSI.brightCyan}║${ANSI.reset}${titleLine}${ANSI.brightCyan}║${ANSI.reset}`,
            `${ANSI.brightCyan}║${emptyLine}║${ANSI.reset}`,
            `${ANSI.brightCyan}║${ANSI.reset}${descLine}${ANSI.brightCyan}║${ANSI.reset}`,
            `${ANSI.brightCyan}║${ANSI.reset}${cmdsLine}${ANSI.brightCyan}║${ANSI.reset}`,
            `${ANSI.brightCyan}╚${topBorder}╝${ANSI.reset}`,
            '',
        ].join('\r\n');

        this.sendOutput(sessionId, welcome + '\r\n');
    }

    private showPrompt(sessionId: string) {
        const internal = this.sessions.get(sessionId);
        if (!internal) return;

        const prompt = `${ANSI.green}user@mock${ANSI.reset}:${ANSI.blue}${internal.cwd}${ANSI.reset}$ `;
        this.sendOutput(sessionId, prompt);
    }

    private executeCommand(sessionId: string, commandLine: string) {
        const internal = this.sessions.get(sessionId);
        if (!internal) return;

        const parts = commandLine.trim().split(/\s+/);
        const command = parts[0].toLowerCase();
        const args = parts.slice(1);

        switch (command) {
            case 'help':
                this.showHelp(sessionId);
                break;
            case 'ls':
                this.executeLs(sessionId, args);
                break;
            case 'pwd':
                this.sendOutput(sessionId, internal.cwd + '\r\n');
                break;
            case 'cd':
                this.executeCd(sessionId, args);
                break;
            case 'echo':
                this.sendOutput(sessionId, args.join(' ') + '\r\n');
                break;
            case 'clear':
            case 'cls':
                this.sendOutput(sessionId, ANSI.clearScreen + ANSI.cursorHome);
                break;
            case 'cat':
                this.executeCat(sessionId, args);
                break;
            case 'whoami':
                this.sendOutput(sessionId, 'user\r\n');
                break;
            case 'date':
                this.sendOutput(sessionId, new Date().toString() + '\r\n');
                break;
            case 'uname':
                this.sendOutput(sessionId, 'MockOS 1.0.0-pure-terminal\r\n');
                break;
            case 'ps':
                this.executePs(sessionId);
                break;
            case 'tree':
                this.executeTree(sessionId);
                break;
            case 'colors':
                this.showColors(sessionId);
                break;
            case 'logo':
                this.showWelcome(sessionId);
                break;
            case 'exit':
                this.sendOutput(sessionId, 'exit\r\n');
                const session = { id: sessionId } as FakeShellSession;
                session.close();
                break;
            case 'vim':
                this.startVim(sessionId, args);
                break;
            default:
                this.sendOutput(sessionId, `${ANSI.brightRed}Command not found: ${command}${ANSI.reset}\r\n`);
                this.sendOutput(sessionId, `${ANSI.dim}Type 'help' for available commands${ANSI.reset}\r\n`);
        }
    }

    private showHelp(sessionId: string) {
        const help = [
            `${ANSI.bright}Available Commands:${ANSI.reset}`,
            '',
            `  ${ANSI.yellow}ls${ANSI.reset} [path]     List directory contents`,
            `  ${ANSI.yellow}pwd${ANSI.reset}          Print working directory`,
            `  ${ANSI.yellow}cd${ANSI.reset} [path]     Change directory`,
            `  ${ANSI.yellow}echo${ANSI.reset} [text]   Print text`,
            `  ${ANSI.yellow}cat${ANSI.reset} [file]    Display file contents`,
            `  ${ANSI.yellow}clear${ANSI.reset}        Clear the screen`,
            `  ${ANSI.yellow}whoami${ANSI.reset}       Display current user`,
            `  ${ANSI.yellow}date${ANSI.reset}         Show current date/time`,
            `  ${ANSI.yellow}uname${ANSI.reset}        Show system information`,
            `  ${ANSI.yellow}ps${ANSI.reset}           List processes`,
            `  ${ANSI.yellow}tree${ANSI.reset}         Show directory tree`,
            `  ${ANSI.yellow}colors${ANSI.reset}       Display color test`,
            `  ${ANSI.yellow}vim${ANSI.reset} [file]    Open vim editor (press 'e' to exit)`,
            `  ${ANSI.yellow}logo${ANSI.reset}         Show welcome logo`,
            `  ${ANSI.yellow}help${ANSI.reset}         Show this help`,
            `  ${ANSI.yellow}exit${ANSI.reset}         Close terminal`,
            '',
        ].join('\r\n');

        this.sendOutput(sessionId, help + '\r\n');
    }

    private executeLs(sessionId: string, args: string[]) {
        const internal = this.sessions.get(sessionId);
        if (!internal) return;

        const path = args[0] || internal.cwd;
        const resolvedPath = this.resolvePath(internal.cwd, path);
        const contents = this.mockFileSystem.get(resolvedPath);

        if (!contents) {
            this.sendOutput(sessionId, `${ANSI.brightRed}ls: cannot access '${path}': No such file or directory${ANSI.reset}\r\n`);
            return;
        }

        const output = contents.map(item => {
            if (item.startsWith('.')) {
                return `${ANSI.dim}${item}${ANSI.reset}`;
            } else if (this.mockFileSystem.has(`${resolvedPath}/${item}`)) {
                return `${ANSI.blue}${ANSI.bright}${item}/${ANSI.reset}`;
            } else if (item.endsWith('.md')) {
                return `${ANSI.brightCyan}${item}${ANSI.reset}`;
            } else if (item.endsWith('.txt')) {
                return `${ANSI.white}${item}${ANSI.reset}`;
            } else {
                return `${ANSI.green}${item}${ANSI.reset}`;
            }
        }).join('  ');

        this.sendOutput(sessionId, output + '\r\n');
    }

    private executeCd(sessionId: string, args: string[]) {
        const internal = this.sessions.get(sessionId);
        if (!internal) return;

        const target = args[0] || '/home/user';
        const newPath = this.resolvePath(internal.cwd, target);

        if (this.mockFileSystem.has(newPath)) {
            internal.cwd = newPath;
        } else {
            this.sendOutput(sessionId, `${ANSI.brightRed}cd: ${target}: No such file or directory${ANSI.reset}\r\n`);
        }
    }

    private executeCat(sessionId: string, args: string[]) {
        if (args.length === 0) {
            this.sendOutput(sessionId, `${ANSI.brightRed}cat: missing file operand${ANSI.reset}\r\n`);
            return;
        }

        const file = args[0];

        const mockFiles: Record<string, string> = {
            'readme.md': `${ANSI.bright}# Project README${ANSI.reset}\r\n\r\nThis is a ${ANSI.green}mock file${ANSI.reset} for testing.`,
            'notes.txt': `${ANSI.yellow}TODO List:${ANSI.reset}\r\n- Test terminal colors\r\n- Verify shell commands\r\n- Check mobile responsiveness`,
            '.bashrc': `${ANSI.cyan}# .bashrc${ANSI.reset}\r\nexport PS1="\\u@\\h:\\w$ "\r\nalias ll='ls -la'`,
        };

        const content = mockFiles[file.toLowerCase()];
        if (content) {
            this.sendOutput(sessionId, content + '\r\n');
        } else {
            this.sendOutput(sessionId, `${ANSI.brightRed}cat: ${file}: No such file or directory${ANSI.reset}\r\n`);
        }
    }

    private executePs(sessionId: string) {
        const processes = [
            `${ANSI.bright}PID    USER     TIME   COMMAND${ANSI.reset}`,
            `  1    root     0:00   init`,
            `  42   user     0:05   ${ANSI.green}bash${ANSI.reset}`,
            `  1337 user     0:01   ${ANSI.cyan}node${ANSI.reset} server.js`,
            `  2048 user     0:00   ${ANSI.yellow}python3${ANSI.reset} app.py`,
            `  31337 user   0:00   ps aux`,
        ].join('\r\n');

        this.sendOutput(sessionId, processes + '\r\n');
    }

    private executeTree(sessionId: string) {
        const tree = [
            `${ANSI.blue}.${ANSI.reset}`,
            `${ANSI.blue}├──${ANSI.reset} ${ANSI.blue}bin/${ANSI.reset}`,
            `${ANSI.blue}├──${ANSI.reset} ${ANSI.blue}etc/${ANSI.reset}`,
            `${ANSI.blue}├──${ANSI.reset} ${ANSI.blue}home/${ANSI.reset}`,
            `${ANSI.blue}│   └──${ANSI.reset} ${ANSI.blue}user/${ANSI.reset}`,
            `${ANSI.blue}│       ├──${ANSI.reset} ${ANSI.blue}Documents/${ANSI.reset}`,
            `${ANSI.blue}│       ├──${ANSI.reset} ${ANSI.blue}Downloads/${ANSI.reset}`,
            `${ANSI.blue}│       ├──${ANSI.reset} ${ANSI.blue}projects/${ANSI.reset}`,
            `${ANSI.blue}│       │   ├──${ANSI.reset} ${ANSI.green}ai-critic${ANSI.reset}`,
            `${ANSI.blue}│       │   └──${ANSI.reset} ${ANSI.green}my-app${ANSI.reset}`,
            `${ANSI.blue}│       └──${ANSI.reset} ${ANSI.yellow}README.md${ANSI.reset}`,
            `${ANSI.blue}├──${ANSI.reset} ${ANSI.blue}usr/${ANSI.reset}`,
            `${ANSI.blue}└──${ANSI.reset} ${ANSI.blue}var/${ANSI.reset}`,
        ].join('\r\n');

        this.sendOutput(sessionId, tree + '\r\n');
    }

    private showColors(sessionId: string) {
        const colors = [
            `${ANSI.bright}Terminal Color Test:${ANSI.reset}`,
            '',
            `  ${ANSI.red}Red${ANSI.reset}     ${ANSI.green}Green${ANSI.reset}     ${ANSI.yellow}Yellow${ANSI.reset}    ${ANSI.blue}Blue${ANSI.reset}`,
            `  ${ANSI.magenta}Magenta${ANSI.reset} ${ANSI.cyan}Cyan${ANSI.reset}      ${ANSI.white}White${ANSI.reset}     ${ANSI.brightRed}Bright Red${ANSI.reset}`,
            `  ${ANSI.brightGreen}Bright Green${ANSI.reset}  ${ANSI.brightYellow}Bright Yellow${ANSI.reset}  ${ANSI.brightBlue}Bright Blue${ANSI.reset}`,
            `  ${ANSI.brightMagenta}Bright Magenta${ANSI.reset}  ${ANSI.brightCyan}Bright Cyan${ANSI.reset}`,
            '',
            `${ANSI.dim}Dim text${ANSI.reset}    ${ANSI.bright}Bright text${ANSI.reset}`,
            '',
        ].join('\r\n');

        this.sendOutput(sessionId, colors + '\r\n');
    }

    private startVim(sessionId: string, args: string[]) {
        const internal = this.sessions.get(sessionId);
        if (!internal) return;

        const filename = args[0] || '[No Name]';
        
        internal.altScreenBuffer = {
            lines: [],
            currentLine: '',
            vimState: { filename, keyLog: [] },
        };

        this.sendOutput(sessionId, ANSI.enterAltScreen + ANSI.clearScreen + ANSI.cursorHome);
        this.renderVimScreen(sessionId);
    }

    private renderVimScreen(sessionId: string) {
        const internal = this.sessions.get(sessionId);
        if (!internal || !internal.altScreenBuffer?.vimState) return;

        const { filename, keyLog } = internal.altScreenBuffer.vimState;
        const { visibleRows, cols } = internal;

        const lines: string[] = [];

        lines.push(`  ${ANSI.green}~${ANSI.reset}`);
        lines.push(`  ${ANSI.green}~${ANSI.reset}  ${ANSI.brightCyan}Mock Vim - Alt Screen Test${ANSI.reset}`);
        lines.push(`  ${ANSI.green}~${ANSI.reset}`);
        lines.push(`  ${ANSI.green}~${ANSI.reset}  This is a minimal vim mock to test alternate screen.`);
        lines.push(`  ${ANSI.green}~${ANSI.reset}`);
        lines.push(`  ${ANSI.green}~${ANSI.reset}  ${ANSI.yellow}Press 'e' to exit vim${ANSI.reset}`);
        lines.push(`  ${ANSI.green}~${ANSI.reset}  Other keys will be logged below.`);
        lines.push(`  ${ANSI.green}~${ANSI.reset}`);
        lines.push(`  ${ANSI.green}~${ANSI.reset}`);

        if (keyLog.length > 0) {
            lines.push(`  ${ANSI.green}~${ANSI.reset}  ${ANSI.dim}Key log (${keyLog.length} keys):${ANSI.reset}`);
            const recentKeys = keyLog.slice(-10);
            const keyDisplay = recentKeys.map(k => {
                if (k === ' ') return '<SP>';
                if (k === '\x1b') return '<ESC>';
                if (k === '\r') return '<CR>';
                if (k === '\n') return '<LF>';
                if (k === '\t') return '<TAB>';
                return k;
            }).join(' ');
            lines.push(`  ${ANSI.green}~${ANSI.reset}  ${ANSI.cyan}${keyDisplay}${ANSI.reset}`);
        }

        while (lines.length < visibleRows - 2) {
            lines.push(`  ${ANSI.green}~${ANSI.reset}`);
        }

        const statusLine = `${ANSI.brightWhite}${ANSI.bgBlue} ${filename.padEnd(cols - 20)} 1,1      All ${ANSI.reset}`;
        lines.push(statusLine);
        lines.push(`${ANSI.brightWhite}${ANSI.bgBlue}-- MOCK VIM --${' '.repeat(Math.max(0, cols - 14))}${ANSI.reset}`);

        internal.outputLines = [];
        internal.currentLine = '';

        this.sendOutput(sessionId, ANSI.cursorHome + lines.join('\r\n'));
    }

    private handleVimInput(sessionId: string, data: string) {
        const internal = this.sessions.get(sessionId);
        if (!internal?.altScreenBuffer?.vimState) return;

        const vimState = internal.altScreenBuffer.vimState;

        for (let i = 0; i < data.length; i++) {
            const char = data[i];

            if (char === 'e' || char === 'E') {
                this.exitVim(sessionId);
                return;
            }

            vimState.keyLog.push(char);
            console.log(`[vim] key pressed: ${JSON.stringify(char)}`);
        }

        this.renderVimScreen(sessionId);
    }

    private exitVim(sessionId: string) {
        const internal = this.sessions.get(sessionId);
        if (!internal?.altScreenBuffer?.vimState) return;

        const keyCount = internal.altScreenBuffer.vimState.keyLog.length;

        this.sendOutput(sessionId, ANSI.exitAltScreen);

        this.sendOutput(sessionId, `${ANSI.dim}[vim] Exited. ${keyCount} keys logged.${ANSI.reset}\r\n`);
        this.showPrompt(sessionId);
    }

    private resolvePath(cwd: string, target: string): string {
        if (target.startsWith('/')) {
            return target;
        }
        if (target === '..') {
            const parts = cwd.split('/').filter(Boolean);
            parts.pop();
            return '/' + parts.join('/');
        }
        if (target === '.') {
            return cwd;
        }
        return cwd === '/' ? `/${target}` : `${cwd}/${target}`;
    }
}

let fakeShellServer: FakeShellServer | null = null;

export function getFakeShellServer(): FakeShellServer {
    if (!fakeShellServer) {
        fakeShellServer = new FakeShellServer();
    }
    return fakeShellServer;
}

export function resetFakeShellServer(): void {
    fakeShellServer = null;
}
