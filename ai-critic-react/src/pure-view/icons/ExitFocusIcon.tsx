export interface ExitFocusIconProps {
    className?: string;
}

export function ExitFocusIcon({ className }: ExitFocusIconProps) {
    return (
        <svg viewBox="0 0 24 24" className={className} aria-hidden="true">
            <polyline points="14 10 21 3" />
            <polyline points="3 21 10 14" />
            <polyline points="21 10 21 3 14 3" />
            <polyline points="3 14 3 21 10 21" />
        </svg>
    );
}
