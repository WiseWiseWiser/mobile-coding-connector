import type { ReactNode, CSSProperties } from 'react';
import './NoZoomingInput.css';

export interface NoZoomingInputProps {
    children: ReactNode;
    className?: string;
    style?: CSSProperties;
}

export function NoZoomingInput({ children, className, style }: NoZoomingInputProps) {
    return (
        <div className={`nozooming-input-wrapper ${className || ''}`} style={style}>
            {children}
        </div>
    );
}
