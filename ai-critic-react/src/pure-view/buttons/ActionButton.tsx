import type { ReactNode, ButtonHTMLAttributes } from 'react';
import './ActionButton.css';

interface ActionButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    children: ReactNode;
}

export function ActionButton({ children, className, ...props }: ActionButtonProps) {
    return (
        <button className={`action-btn ${className || ''}`} {...props}>
            {children}
        </button>
    );
}
