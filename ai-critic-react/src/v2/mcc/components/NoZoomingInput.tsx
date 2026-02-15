import type { ReactNode, CSSProperties } from 'react';

export interface NoZoomingInputProps {
    children: ReactNode;
    className?: string;
    style?: CSSProperties;
}

const noZoomStyle: CSSProperties = {
    fontSize: '16px',
    touchAction: 'manipulation',
};

export function NoZoomingInput({ children, className, style }: NoZoomingInputProps) {
    return (
        <div className={className} style={{ ...noZoomStyle, ...style }}>
            {children}
        </div>
    );
}
