interface OpenInNewIconProps {
    className?: string;
}

export function OpenInNewIcon({ className }: OpenInNewIconProps) {
    return (
        <svg
            className={className}
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
        >
            <path d="M14 3h7v7" />
            <path d="M10 14L21 3" />
            <path d="M21 14v7H3V3h7" />
        </svg>
    );
}
