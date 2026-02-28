export interface EnterFocusIconProps {
    className?: string;
}

export function EnterFocusIcon({ className }: EnterFocusIconProps) {
    return (
        <svg viewBox="0 0 24 24" className={className} aria-hidden="true">
            <polyline points="15 3 21 3 21 9" />
            <polyline points="9 21 3 21 3 15" />
            <polyline points="21 15 21 21 15 21" />
            <polyline points="3 9 3 3 9 3" />
        </svg>
    );
}
