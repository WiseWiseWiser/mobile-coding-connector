import { useStreamingAction } from '../hooks/useStreamingAction';
import type { StreamingActionResult } from '../hooks/useStreamingAction';
import { StreamingButton, StreamingLogs } from './StreamingComponents';
import './StreamingActionButton.css';

export type { StreamingActionResult };

export interface StreamingActionButtonProps {
    /** Button label when idle */
    label: string;
    /** Button label when running */
    runningLabel: string;
    /** Function that returns a Response with SSE stream */
    action: () => Promise<Response>;
    /** Optional callback when action completes */
    onComplete?: (result: StreamingActionResult) => void;
    /** Additional CSS class for the button */
    className?: string;
    /** Whether the button is disabled */
    disabled?: boolean;
    /** Max height for log viewer. Default: 150 */
    logMaxHeight?: number;
    /** Icon to show in button (optional) */
    icon?: React.ReactNode;
}

/** Combined streaming action button with integrated log display. */
export function StreamingActionButton({
    label,
    runningLabel,
    action,
    onComplete,
    className = '',
    disabled = false,
    logMaxHeight = 150,
    icon,
}: StreamingActionButtonProps) {
    const [state, controls] = useStreamingAction(onComplete);

    const handleClick = () => {
        controls.run(action);
    };

    return (
        <div className="streaming-action-button-container">
            <StreamingButton
                label={label}
                runningLabel={runningLabel}
                onClick={handleClick}
                running={state.running}
                className={className}
                disabled={disabled}
                icon={icon}
            />
            <StreamingLogs
                state={state}
                pendingMessage={runningLabel}
                maxHeight={logMaxHeight}
            />
        </div>
    );
}
