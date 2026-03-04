import './ErrorMessage.css';

export interface ErrorMessageProps {
    children: React.ReactNode;
    className?: string;
}

export function ErrorMessage({ children, className }: ErrorMessageProps) {
    return (
        <div className={className ? `pure-error-message ${className}` : 'pure-error-message'}>
            {children}
        </div>
    );
}
