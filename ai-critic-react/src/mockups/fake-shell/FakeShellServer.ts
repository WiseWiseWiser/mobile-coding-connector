// FakeShellServer.ts - A pure frontend shell server that simulates terminal responses

// ANSI color codes for terminal output
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
};

export interface FakeShellSession {
    id: string;
    cwd: string;
    cols: number;
    rows: number;
    history: string[];
    send: (data: string) => void;
    onData: (callback: (data: string) => void) => () => void;
    onClose: (callback: () => void) => () => void;
    resize: (cols: number, rows: number) => void;
    close: () => void;
}



/**
 * FakeShellServer - A pure frontend shell server that simulates terminal responses.
 * This runs entirely in the browser and provides realistic shell behavior with colors.
 */
export class FakeShellServer {
    private sessions: Map<string, FakeShellSession> = new Map();
    private sessionCounter = 0;
    private mockFileSystem: Map<string, string[]> = new Map();

    constructor() {
        // Initialize mock filesystem
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
        const dataCallbacks: ((data: string) => void)[] = [];
        const closeCallbacks: (() => void)[] = [];

        const session: FakeShellSession = {
            id: sessionId,
            cwd: initialCwd,
            cols: 80,
            rows: 24,
            history: [],

            send: (data: string) => {
                // Handle the input data character by character
                for (let i = 0; i < data.length; i++) {
                    const char = data[i];
                    
                    if (char === '\r' || char === '\n') {
                        // User pressed Enter - execute command
                        // First move to new line
                        dataCallbacks.forEach(cb => cb('\r\n'));
                        
                        // Get the command
                        const command = session.history.join('').trim();
                        session.history = [];
                        
                        // Execute if there's a command
                        if (command) {
                            this.executeCommand(session, command, dataCallbacks);
                        }
                        
                        // Show prompt
                        this.showPrompt(session, dataCallbacks);
                    } else if (char === '\x7f' || char === '\b') {
                        // Backspace
                        if (session.history.length > 0) {
                            session.history.pop();
                            dataCallbacks.forEach(cb => cb('\b \b'));
                        }
                    } else if (char === '\x03') {
                        // Ctrl+C - cancel current input
                        session.history = [];
                        dataCallbacks.forEach(cb => cb('^C'));
                        dataCallbacks.forEach(cb => cb('\r\n'));
                        this.showPrompt(session, dataCallbacks);
                    } else if (char === '\x0c') {
                        // Ctrl+L (clear screen)
                        dataCallbacks.forEach(cb => cb('\x1b[2J\x1b[H'));
                        this.showPrompt(session, dataCallbacks);
                    } else if (char === '\t') {
                        // Tab - insert spaces
                        session.history.push('    ');
                        dataCallbacks.forEach(cb => cb('    '));
                    } else {
                        // Regular character - echo it and add to history
                        session.history.push(char);
                        dataCallbacks.forEach(cb => cb(char));
                    }
                }
            },

            onData: (callback: (data: string) => void) => {
                dataCallbacks.push(callback);
                return () => {
                    const index = dataCallbacks.indexOf(callback);
                    if (index > -1) dataCallbacks.splice(index, 1);
                };
            },

            onClose: (callback: () => void) => {
                closeCallbacks.push(callback);
                return () => {
                    const index = closeCallbacks.indexOf(callback);
                    if (index > -1) closeCallbacks.splice(index, 1);
                };
            },

            resize: (cols: number, rows: number) => {
                session.cols = cols;
                session.rows = rows;
            },

            close: () => {
                closeCallbacks.forEach(cb => cb());
                this.sessions.delete(sessionId);
            },
        };

        this.sessions.set(sessionId, session);

        // Send initial welcome message and prompt
        setTimeout(() => {
            this.showWelcome(session, dataCallbacks);
            this.showPrompt(session, dataCallbacks);
        }, 100);

        return session;
    }

    private showWelcome(session: FakeShellSession, callbacks: ((data: string) => void)[]) {
        const width = Math.min(session.cols, 100);
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

        callbacks.forEach(cb => cb(welcome + '\r\n'));
    }

    private showPrompt(session: FakeShellSession, callbacks: ((data: string) => void)[]) {
        const prompt = `${ANSI.green}user@mock${ANSI.reset}:${ANSI.blue}${session.cwd}${ANSI.reset}$ `;
        callbacks.forEach(cb => cb(prompt));
    }

    private executeCommand(
        session: FakeShellSession, 
        commandLine: string, 
        callbacks: ((data: string) => void)[]
    ) {
        const parts = commandLine.trim().split(/\s+/);
        const command = parts[0].toLowerCase();
        const args = parts.slice(1);

        switch (command) {
            case 'help':
                this.showHelp(callbacks);
                break;
            case 'ls':
                this.executeLs(session, args, callbacks);
                break;
            case 'pwd':
                callbacks.forEach(cb => cb(session.cwd + '\r\n'));
                break;
            case 'cd':
                this.executeCd(session, args, callbacks);
                break;
            case 'echo':
                callbacks.forEach(cb => cb(args.join(' ') + '\r\n'));
                break;
            case 'clear':
            case 'cls':
                callbacks.forEach(cb => cb('\x1b[2J\x1b[H'));
                break;
            case 'cat':
                this.executeCat(session, args, callbacks);
                break;
            case 'whoami':
                callbacks.forEach(cb => cb('user\r\n'));
                break;
            case 'date':
                callbacks.forEach(cb => cb(new Date().toString() + '\r\n'));
                break;
            case 'uname':
                callbacks.forEach(cb => cb('MockOS 1.0.0-pure-terminal\r\n'));
                break;
            case 'ps':
                this.executePs(callbacks);
                break;
            case 'tree':
                this.executeTree(session, callbacks);
                break;
            case 'colors':
                this.showColors(callbacks);
                break;
            case 'logo':
                this.showWelcome(session, callbacks);
                break;
            case 'exit':
                callbacks.forEach(cb => cb('exit\r\n'));
                session.close();
                break;
            default:
                callbacks.forEach(cb => 
                    cb(`${ANSI.brightRed}Command not found: ${command}${ANSI.reset}\r\n`)
                );
                callbacks.forEach(cb => 
                    cb(`${ANSI.dim}Type 'help' for available commands${ANSI.reset}\r\n`)
                );
        }
    }

    private showHelp(callbacks: ((data: string) => void)[]) {
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
            `  ${ANSI.yellow}logo${ANSI.reset}         Show welcome logo`,
            `  ${ANSI.yellow}help${ANSI.reset}         Show this help`,
            `  ${ANSI.yellow}exit${ANSI.reset}         Close terminal`,
            '',
        ].join('\r\n');

        callbacks.forEach(cb => cb(help + '\r\n'));
    }

    private executeLs(
        session: FakeShellSession, 
        args: string[], 
        callbacks: ((data: string) => void)[]
    ) {
        const path = args[0] || session.cwd;
        const resolvedPath = this.resolvePath(session.cwd, path);
        const contents = this.mockFileSystem.get(resolvedPath);

        if (!contents) {
            callbacks.forEach(cb => 
                cb(`${ANSI.brightRed}ls: cannot access '${path}': No such file or directory${ANSI.reset}\r\n`)
            );
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

        callbacks.forEach(cb => cb(output + '\r\n'));
    }

    private executeCd(
        session: FakeShellSession, 
        args: string[], 
        callbacks: ((data: string) => void)[]
    ) {
        const target = args[0] || '/home/user';
        const newPath = this.resolvePath(session.cwd, target);

        if (this.mockFileSystem.has(newPath)) {
            session.cwd = newPath;
        } else {
            callbacks.forEach(cb => 
                cb(`${ANSI.brightRed}cd: ${target}: No such file or directory${ANSI.reset}\r\n`)
            );
        }
    }

    private executeCat(
        _session: FakeShellSession, 
        args: string[], 
        callbacks: ((data: string) => void)[]
    ) {
        if (args.length === 0) {
            callbacks.forEach(cb => cb(`${ANSI.brightRed}cat: missing file operand${ANSI.reset}\r\n`));
            return;
        }

        const file = args[0];
        
        // Simulate file contents
        const mockFiles: Record<string, string> = {
            'readme.md': `${ANSI.bright}# Project README${ANSI.reset}\r\n\r\nThis is a ${ANSI.green}mock file${ANSI.reset} for testing.`,
            'notes.txt': `${ANSI.yellow}TODO List:${ANSI.reset}\r\n- Test terminal colors\r\n- Verify shell commands\r\n- Check mobile responsiveness`,
            '.bashrc': `${ANSI.cyan}# .bashrc${ANSI.reset}\r\nexport PS1="\\u@\\h:\\w$ "\r\nalias ll='ls -la'`,
        };

        const content = mockFiles[file.toLowerCase()];
        if (content) {
            callbacks.forEach(cb => cb(content + '\r\n'));
        } else {
            callbacks.forEach(cb => 
                cb(`${ANSI.brightRed}cat: ${file}: No such file or directory${ANSI.reset}\r\n`)
            );
        }
    }

    private executePs(callbacks: ((data: string) => void)[]) {
        const processes = [
            `${ANSI.bright}PID    USER     TIME   COMMAND${ANSI.reset}`,
            `  1    root     0:00   init`,
            `  42   user     0:05   ${ANSI.green}bash${ANSI.reset}`,
            `  1337 user     0:01   ${ANSI.cyan}node${ANSI.reset} server.js`,
            `  2048 user     0:00   ${ANSI.yellow}python3${ANSI.reset} app.py`,
            `  31337 user   0:00   ps aux`,
        ].join('\r\n');

        callbacks.forEach(cb => cb(processes + '\r\n'));
    }

    private executeTree(
        _session: FakeShellSession, 
        callbacks: ((data: string) => void)[]
    ) {
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

        callbacks.forEach(cb => cb(tree + '\r\n'));
    }

    private showColors(callbacks: ((data: string) => void)[]) {
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

        callbacks.forEach(cb => cb(colors + '\r\n'));
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

// Singleton instance
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
