import type { RefObject } from 'react';
import type { StreamEvent } from '../../mockups/fake';
import type { DriverAgentStatus } from './types';
import './DriverSection.css';

interface DriverSectionProps {
    status: DriverAgentStatus;
    progress: number;
    streamEvents: StreamEvent[];
    streamEventsRef: RefObject<HTMLDivElement | null>;
    onStart: () => void;
    onPause: () => void;
    onResume: () => void;
    onAbort: () => void;
}

export function DriverSection({
    status,
    progress,
    streamEvents,
    streamEventsRef,
    onStart,
    onPause,
    onResume,
    onAbort,
}: DriverSectionProps) {
    return (
        <div className="fm-run-section">
            <div className="fm-run-controls">
                <div className="fm-driver-control">
                    {status === 'idle' && (
                        <button className="fm-driver-btn fm-driver-start" onClick={onStart}>
                            ▶ Start the driver agent to implement the feature
                        </button>
                    )}
                    {status === 'running' && (
                        <button className="fm-driver-btn fm-driver-pause" onClick={onPause}>
                            ⏸ Pause
                        </button>
                    )}
                    {status === 'paused' && (
                        <div className="fm-driver-paused-controls">
                            <button className="fm-driver-btn fm-driver-resume" onClick={onResume}>
                                ▶ Resume
                            </button>
                            <button className="fm-driver-btn fm-driver-abort" onClick={onAbort}>
                                ⏹ Abort
                            </button>
                        </div>
                    )}
                    {status === 'finished' && (
                        <div className="fm-driver-finished">
                            <span className="fm-finished-badge">✓ Finished</span>
                        </div>
                    )}
                </div>
            </div>
            {(status === 'running' || status === 'paused' || status === 'finished') && (
                <div className="fm-run-progress">
                    <div className="fm-progress-bar">
                        <div className="fm-progress-fill" style={{ width: `${progress}%` }}></div>
                    </div>
                    <div className="fm-stream-events" ref={streamEventsRef}>
                        {streamEvents.map((event, i) => (
                            <div key={i} className={`fm-stream-event fm-event-type-${event.type}`}>
                                {event.step && <span className="fm-event-step">[{event.step}]</span>}
                                <span className="fm-event-message">{event.message}</span>
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
