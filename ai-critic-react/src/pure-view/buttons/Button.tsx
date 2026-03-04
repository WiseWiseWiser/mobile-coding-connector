import './Button.css';

export const ButtonVariants = {
    Primary: 'primary',
    Secondary: 'secondary',
    Cancel: 'cancel',
    Danger: 'danger',
    Link: 'link',
    Start: 'start',
    Stop: 'stop',
} as const;

export type ButtonVariant = typeof ButtonVariants[keyof typeof ButtonVariants];

export interface ButtonProps {
    variant?: ButtonVariant;
    children: React.ReactNode;
    onClick?: (e: React.MouseEvent) => void;
    disabled?: boolean;
    className?: string;
    title?: string;
    type?: 'button' | 'submit' | 'reset';
}

export function Button({
    variant = 'primary',
    children,
    onClick,
    disabled,
    className,
    title,
    type = 'button',
}: ButtonProps) {
    const cls = className
        ? `pure-btn pure-btn--${variant} ${className}`
        : `pure-btn pure-btn--${variant}`;
    return (
        <button
            className={cls}
            onClick={onClick}
            disabled={disabled}
            title={title}
            type={type}
        >
            {children}
        </button>
    );
}
