export type TerminalConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'exited' | 'unavailable';

interface ConnectionOptions {
    sessionId?: string;
    initialCommand?: string;
    url: (sessionId?: string) => string;
    onState: (state: TerminalConnectionState) => void;
    onSessionId: (id: string) => void;
    onOutput: (data: string | Uint8Array) => void;
    onReplayStart: () => void;
    createSocket?: (url: string) => WebSocket;
}

/** Transport lifetime is independent of the terminal renderer and the PTY.
 * Reconnects attach to the assigned session; only a server lifecycle message
 * may declare it exited/missing. Socket generations fence late callbacks.
 */
export class TerminalConnection {
    private options: ConnectionOptions;
    private socket: WebSocket | null = null;
    private sessionId?: string;
    private generation = 0;
    private disposed = false;
    private terminal = false;
    private attempt = 0;
    private commandSent = false;
    private restoring: boolean;
    private retryTimer?: ReturnType<typeof setTimeout>;
    private deadlineTimer?: ReturnType<typeof setTimeout>;
    private heartbeatTimer?: ReturnType<typeof setTimeout>;
    private state: TerminalConnectionState = 'connecting';

    constructor(options: ConnectionOptions) {
        this.options = options;
        this.sessionId = options.sessionId;
        this.restoring = !!options.sessionId;
    }

    start() {
        if (this.disposed || this.terminal) return;
        this.clearTimers();
        this.invalidateSocket();
        const generation = this.generation;
        this.setState(this.sessionId || this.attempt ? 'reconnecting' : 'connecting');
        let socket: WebSocket;
        try {
            socket = (this.options.createSocket ?? (url => new WebSocket(url)))(this.options.url(this.sessionId));
        } catch {
            this.scheduleRetry();
            return;
        }
        this.socket = socket;
        socket.binaryType = 'arraybuffer';
        const current = () => !this.disposed && generation === this.generation;
        // Bounds both connection establishment and waiting for the session handshake.
        this.deadlineTimer = setTimeout(() => { if (current()) this.retry(); }, 10000);
        socket.onopen = () => {
            // OPEN says nothing about whether the shell is still running.
            if (!current()) return;
        };
        socket.onmessage = event => {
            if (!current()) return;
            if (event.data instanceof ArrayBuffer) {
                this.options.onOutput(new Uint8Array(event.data));
                return;
            }
            const text = String(event.data);
            let msg: Record<string, unknown>;
            try {
                msg = JSON.parse(text);
            } catch {
                this.options.onOutput(text);
                return;
            }
            if (!msg || typeof msg !== 'object' || !msg.type) {
                this.options.onOutput(text);
                return;
            }
            switch (msg.type) {
                case 'session_id':
                    if (typeof msg.session_id === 'string') {
                        this.sessionId = msg.session_id;
                        this.options.onSessionId(msg.session_id);
                        // The server replays the session on every attachment. Reset
                        // only once a replacement attachment has actually arrived.
                        this.options.onReplayStart();
                    }
                    break;
                case 'session_state':
                    if (msg.status === 'running') {
                        this.clearDeadline();
                        this.attempt = 0;
                        this.setState('connected');
                        if (!this.restoring && !this.commandSent && this.options.initialCommand) {
                            this.commandSent = true;
                            this.send(this.options.initialCommand + '\n');
                        }
                        this.restoring = true;
                        this.armHeartbeat();
                    } else if (msg.status === 'exited' || msg.status === 'missing') {
                        this.terminal = true;
                        this.clearTimers();
                        this.setState(msg.status === 'exited' ? 'exited' : 'unavailable');
                        this.invalidateSocket();
                    }
                    break;
                case 'pong':
                    this.clearDeadline();
                    this.armHeartbeat();
                    break;
                case 'error':
                    this.options.onOutput('\r\n' + String(msg.message ?? 'Terminal error') + '\r\n');
                    break;
                // attach_role and future controls are metadata, never PTY output.
            }
        };
        socket.onerror = () => { if (current() && !this.terminal) this.retry(); };
        socket.onclose = () => { if (current() && !this.terminal) this.retry(); };
    }

    send(data: string) {
        if (this.state === 'connected' && this.socket?.readyState === 1) {
            this.socket.send(data);
        }
    }

    /** Replace even an apparently OPEN socket on resume (mobile networks can
     * leave it half-open). Web attachments cannot reap each other's PTY. */
    reconnect() {
        // Coalesce pageshow/visibility/online bursts while the handshake is in
        // flight, especially before a new session has received its identity.
        if (this.socket && this.state !== 'connected' && this.socket.readyState < 2) return;
        if (!this.disposed && !this.terminal) this.start();
    }

    dispose() {
        this.disposed = true;
        this.clearTimers();
        this.invalidateSocket();
    }

    private setState(state: TerminalConnectionState) {
        this.state = state;
        this.options.onState(state);
    }

    private retry() {
        this.clearTimers();
        this.invalidateSocket();
        this.setState('reconnecting');
        this.scheduleRetry();
    }

    private scheduleRetry() {
        const delay = Math.min(500 * 2 ** Math.min(this.attempt++, 6), 15000);
        this.retryTimer = setTimeout(() => this.start(), delay);
    }

    private armHeartbeat() {
        clearTimeout(this.heartbeatTimer);
        this.heartbeatTimer = setTimeout(() => {
            if (this.socket?.readyState !== 1) { this.retry(); return; }
            this.socket.send(JSON.stringify({ type: 'ping' }));
            this.deadlineTimer = setTimeout(() => this.retry(), 5000);
        }, 25000);
    }

    private clearDeadline() {
        clearTimeout(this.deadlineTimer);
        this.deadlineTimer = undefined;
    }

    private clearTimers() {
        clearTimeout(this.retryTimer);
        clearTimeout(this.heartbeatTimer);
        this.clearDeadline();
    }

    private invalidateSocket() {
        ++this.generation;
        const socket = this.socket;
        this.socket = null;
        if (socket) {
            // Leave session deletion to explicit REST DELETE. In particular no
            // "unused session" heuristic on refresh, cleanup, or network failure.
            socket.onopen = socket.onmessage = socket.onerror = socket.onclose = null;
            socket.close();
        }
    }
}
