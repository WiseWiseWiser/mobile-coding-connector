import { useState, useEffect, useRef } from 'react';
import './TransferProgress.css';

function formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    const size = bytes / Math.pow(1024, i);
    return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

function formatSpeed(bytesPerSecond: number): string {
    return formatFileSize(bytesPerSecond) + '/s';
}

export interface TransferProgressData {
    loaded: number;
    total: number;
    percent: number;
}

interface TransferProgressProps {
    progress: TransferProgressData | null;
    label?: string;
}

export function TransferProgress({ progress, label = 'Transfer' }: TransferProgressProps) {
    const startTimeRef = useRef<number>(0);
    const lastSampleRef = useRef<{ time: number; loaded: number }>({ time: 0, loaded: 0 });
    const [avgSpeed, setAvgSpeed] = useState(0);
    const [currentSpeed, setCurrentSpeed] = useState(0);

    useEffect(() => {
        if (!progress) {
            startTimeRef.current = 0;
            lastSampleRef.current = { time: 0, loaded: 0 };
            setAvgSpeed(0);
            setCurrentSpeed(0);
            return;
        }

        const now = Date.now();

        // Initialize on first progress event
        if (startTimeRef.current === 0) {
            startTimeRef.current = now;
            lastSampleRef.current = { time: now, loaded: progress.loaded };
            return;
        }

        // Calculate average speed
        const elapsedSec = (now - startTimeRef.current) / 1000;
        if (elapsedSec > 0) {
            setAvgSpeed(progress.loaded / elapsedSec);
        }

        // Calculate current speed (using last sample, min 500ms interval)
        const lastSample = lastSampleRef.current;
        const sampleElapsed = now - lastSample.time;
        if (sampleElapsed >= 500) {
            const bytesDelta = progress.loaded - lastSample.loaded;
            const secDelta = sampleElapsed / 1000;
            if (secDelta > 0) {
                setCurrentSpeed(bytesDelta / secDelta);
            }
            lastSampleRef.current = { time: now, loaded: progress.loaded };
        }
    }, [progress]);

    if (!progress) return null;

    return (
        <div className="transfer-progress">
            <div className="transfer-progress-bar-bg">
                <div className="transfer-progress-bar" style={{ width: `${progress.percent}%` }} />
            </div>
            <div className="transfer-progress-info">
                <span className="transfer-progress-percent">
                    {label}: {progress.percent}%
                </span>
                <span className="transfer-progress-size">
                    {formatFileSize(progress.loaded)} / {formatFileSize(progress.total)}
                </span>
            </div>
            <div className="transfer-progress-speed">
                <span>Avg: {formatSpeed(avgSpeed)}</span>
                <span>Current: {formatSpeed(currentSpeed)}</span>
            </div>
        </div>
    );
}
