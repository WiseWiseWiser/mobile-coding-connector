import { useState, useEffect, useRef } from 'react';
import { UploadPhases } from '../../../api/fileupload';
import type { UploadPhase } from '../../../api/fileupload';
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
    /** Current upload phase */
    phase?: UploadPhase;
    /** Current chunk index (0-based) */
    chunkIndex?: number;
    /** Total number of chunks */
    totalChunks?: number;
    /** Bytes loaded within the current chunk */
    chunkLoaded?: number;
    /** Total bytes in the current chunk */
    chunkTotal?: number;
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

    const isMerging = progress.phase === UploadPhases.Merging;
    const hasChunkInfo = progress.totalChunks != null && progress.totalChunks > 1;
    const chunkPercent = (progress.chunkLoaded != null && progress.chunkTotal)
        ? Math.round((progress.chunkLoaded / progress.chunkTotal) * 100)
        : 100;

    return (
        <div className="transfer-progress">
            {/* Overall progress bar */}
            <div className="transfer-progress-bar-bg">
                <div className="transfer-progress-bar" style={{ width: `${progress.percent}%` }} />
            </div>
            <div className="transfer-progress-info">
                <span className="transfer-progress-percent">
                    {isMerging ? 'Merging...' : `${label}: ${progress.percent}%`}
                </span>
                <span className="transfer-progress-size">
                    {formatFileSize(progress.loaded)} / {formatFileSize(progress.total)}
                </span>
            </div>

            {/* Chunk-level progress */}
            {hasChunkInfo && !isMerging && (
                <div className="transfer-progress-chunk">
                    <div className="transfer-progress-chunk-label">
                        Chunk {(progress.chunkIndex ?? 0) + 1} / {progress.totalChunks}
                    </div>
                    <div className="transfer-progress-chunk-bar-bg">
                        <div className="transfer-progress-chunk-bar" style={{ width: `${chunkPercent}%` }} />
                    </div>
                </div>
            )}

            {/* Merging indicator */}
            {isMerging && (
                <div className="transfer-progress-merging">
                    Combining chunks on server...
                </div>
            )}

            {/* Speed info */}
            {!isMerging && (
                <div className="transfer-progress-speed">
                    <span>Avg: {formatSpeed(avgSpeed)}</span>
                    <span>Current: {formatSpeed(currentSpeed)}</span>
                </div>
            )}
        </div>
    );
}
