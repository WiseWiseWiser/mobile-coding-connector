import './button-common.css';
import './TestButton.css';

interface TestButtonProps {
    onClick: () => void;
    disabled?: boolean;
    running?: boolean;
    className?: string;
    label?: string;
    runningLabel?: string;
}

export function TestButton({
    onClick,
    disabled,
    running,
    className,
    label = 'Test',
    runningLabel = 'Testing...',
}: TestButtonProps) {
    return (
        <button
            className={`btn-base test-btn${className ? ` ${className}` : ''}`}
            onClick={onClick}
            disabled={disabled || running}
        >
            {running ? runningLabel : label}
        </button>
    );
}
