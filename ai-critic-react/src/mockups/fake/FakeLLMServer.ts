export interface StreamEvent {
    type: 'start' | 'step' | 'done' | 'aborted';
    step?: string;
    message?: string;
    progress?: number;
    output?: string;
    timestamp: number;
}

interface SessionInternal {
    id: string;
    aborted: boolean;
    events: StreamEvent[];
}

export interface FakeLLMSession {
    id: string;
    onStart: (callback: () => void) => () => void;
    onStep: (callback: (event: StreamEvent) => void) => () => void;
    onDone: (callback: (event: StreamEvent) => void) => () => void;
    onAborted: (callback: (event: StreamEvent) => void) => () => void;
    abort: () => void;
    getEvents: () => StreamEvent[];
}

const stepResponses = [
    { step: 'understanding', message: 'Parsing feature request and identifying key requirements...' },
    { step: 'understanding', message: 'Found 6 key requirements: authentication, CRUD, upvoting, WebSocket, push notifications, admin dashboard' },
    { step: 'clarifying', message: 'Consulting Architect Agent for design decisions...' },
    { step: 'clarifying', message: 'Database: PostgreSQL with Prisma ORM selected' },
    { step: 'clarifying', message: 'API: RESTful with JSON response format' },
    { step: 'clarifying', message: 'Real-time: Socket.io for WebSocket communication' },
    { step: 'implementing', message: 'Delegating to Coder Agent...' },
    { step: 'implementing', message: 'Creating database schema for feature requests...' },
    { step: 'implementing', message: 'Implementing REST API endpoints...' },
    { step: 'implementing', message: 'Adding WebSocket integration...' },
    { step: 'implementing', message: 'Building frontend components...' },
    { step: 'verifying', message: 'Running unit tests...' },
    { step: 'verifying', message: 'Running integration tests...' },
    { step: 'verifying', message: 'Checking code coverage: 87%' },
    { step: 'verifying', message: 'All checks passed!' },
];

let sessionCounter = 0;

class FakeLLMServerImpl {
    private sessions: Map<string, SessionInternal> = new Map();
    private startCallbacks: Map<string, (() => void)[]> = new Map();
    private stepCallbacks: Map<string, ((event: StreamEvent) => void)[]> = new Map();
    private doneCallbacks: Map<string, ((event: StreamEvent) => void)[]> = new Map();
    private abortedCallbacks: Map<string, ((event: StreamEvent) => void)[]> = new Map();

    startStream(_prompt: string, _projectPath?: string): FakeLLMSession {
        const sessionId = `fake-llm-${++sessionCounter}`;
        
        const internal: SessionInternal = {
            id: sessionId,
            aborted: false,
            events: [],
        };
        
        this.sessions.set(sessionId, internal);

        const emit = (event: StreamEvent) => {
            internal.events.push(event);
            
            if (event.type === 'start') {
                this.startCallbacks.get(sessionId)?.forEach(cb => cb());
            } else if (event.type === 'step') {
                this.stepCallbacks.get(sessionId)?.forEach(cb => cb(event));
            } else if (event.type === 'done') {
                this.doneCallbacks.get(sessionId)?.forEach(cb => cb(event));
            } else if (event.type === 'aborted') {
                this.abortedCallbacks.get(sessionId)?.forEach(cb => cb(event));
            }
        };

        setTimeout(() => {
            if (internal.aborted) return;
            emit({ type: 'start', message: 'Starting driver agent...', timestamp: Date.now() });

            let stepIndex = 0;
            const runNextStep = () => {
                if (internal.aborted) {
                    emit({ type: 'aborted', message: 'Stream stopped by user', timestamp: Date.now() });
                    return;
                }

                if (stepIndex >= stepResponses.length) {
                    emit({
                        type: 'done',
                        message: 'Feature implementation complete!',
                        output: 'Successfully implemented: authentication, CRUD operations, upvoting system, WebSocket real-time updates, push notifications, and admin dashboard.',
                        timestamp: Date.now(),
                    });
                    return;
                }

                const step = stepResponses[stepIndex];
                const progress = Math.round(((stepIndex + 1) * 100) / stepResponses.length);
                
                emit({
                    type: 'step',
                    step: step.step,
                    message: step.message,
                    progress,
                    timestamp: Date.now(),
                });

                stepIndex++;
                const delay = Math.random() * 400 + 200;
                setTimeout(runNextStep, delay);
            };

            runNextStep();
        }, 100);

        return {
            id: sessionId,
            onStart: (callback) => {
                if (!this.startCallbacks.has(sessionId)) {
                    this.startCallbacks.set(sessionId, []);
                }
                this.startCallbacks.get(sessionId)!.push(callback);
                return () => {
                    const cbs = this.startCallbacks.get(sessionId);
                    if (cbs) {
                        const idx = cbs.indexOf(callback);
                        if (idx >= 0) cbs.splice(idx, 1);
                    }
                };
            },
            onStep: (callback) => {
                if (!this.stepCallbacks.has(sessionId)) {
                    this.stepCallbacks.set(sessionId, []);
                }
                this.stepCallbacks.get(sessionId)!.push(callback);
                return () => {
                    const cbs = this.stepCallbacks.get(sessionId);
                    if (cbs) {
                        const idx = cbs.indexOf(callback);
                        if (idx >= 0) cbs.splice(idx, 1);
                    }
                };
            },
            onDone: (callback) => {
                if (!this.doneCallbacks.has(sessionId)) {
                    this.doneCallbacks.set(sessionId, []);
                }
                this.doneCallbacks.get(sessionId)!.push(callback);
                return () => {
                    const cbs = this.doneCallbacks.get(sessionId);
                    if (cbs) {
                        const idx = cbs.indexOf(callback);
                        if (idx >= 0) cbs.splice(idx, 1);
                    }
                };
            },
            onAborted: (callback) => {
                if (!this.abortedCallbacks.has(sessionId)) {
                    this.abortedCallbacks.set(sessionId, []);
                }
                this.abortedCallbacks.get(sessionId)!.push(callback);
                return () => {
                    const cbs = this.abortedCallbacks.get(sessionId);
                    if (cbs) {
                        const idx = cbs.indexOf(callback);
                        if (idx >= 0) cbs.splice(idx, 1);
                    }
                };
            },
            abort: () => {
                internal.aborted = true;
            },
            getEvents: () => [...internal.events],
        };
    }

    stop(sessionId: string): void {
        const session = this.sessions.get(sessionId);
        if (session) {
            session.aborted = true;
        }
    }
}

let serverInstance: FakeLLMServerImpl | null = null;

export function getFakeLLMServer(): FakeLLMServerImpl {
    if (!serverInstance) {
        serverInstance = new FakeLLMServerImpl();
    }
    return serverInstance;
}

export { FakeLLMServerImpl as FakeLLMServer };
