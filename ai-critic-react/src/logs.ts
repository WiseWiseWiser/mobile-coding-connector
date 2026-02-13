import { useState, useCallback } from 'react';

export type LogLevel = 'log' | 'warn' | 'error' | 'info' | 'debug';

export interface LogEntry {
    id: number;
    timestamp: Date;
    level: LogLevel;
    message: string;
    args: unknown[];
}

let logId = 0;
const maxLogs = 1000;

const logStore: LogEntry[] = [];

const originalConsole = {
    log: console.log,
    warn: console.warn,
    error: console.error,
    info: console.info,
    debug: console.debug,
};

function createInterceptor(level: LogLevel) {
    return (...args: unknown[]) => {
        const message = args.map(arg => {
            if (typeof arg === 'object') {
                try {
                    return JSON.stringify(arg);
                } catch {
                    return String(arg);
                }
            }
            return String(arg);
        }).join(' ');

        const entry: LogEntry = {
            id: ++logId,
            timestamp: new Date(),
            level,
            message,
            args,
        };

        logStore.push(entry);
        if (logStore.length > maxLogs) {
            logStore.shift();
        }

        originalConsole[level].apply(console, args);
    };
}

if (typeof window !== 'undefined') {
    console.log = createInterceptor('log');
    console.warn = createInterceptor('warn');
    console.error = createInterceptor('error');
    console.info = createInterceptor('info');
    console.debug = createInterceptor('debug');
}

export function getLogs(): LogEntry[] {
    return [...logStore];
}

export function clearLogs(): void {
    logStore.length = 0;
}

export function useLogs() {
    const [logs, setLogs] = useState<LogEntry[]>([]);

    const refreshLogs = useCallback(() => {
        setLogs(getLogs());
    }, []);

    return { logs, refreshLogs, clearLogs };
}

export function getLogsAsText(): string {
    return logStore.map(entry => {
        const time = entry.timestamp.toISOString();
        const level = entry.level.toUpperCase().padEnd(5);
        return `[${time}] ${level} ${entry.message}`;
    }).join('\n');
}
