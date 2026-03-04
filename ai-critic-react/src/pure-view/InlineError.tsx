import './InlineError.css';

export interface InlineErrorProps {
    children: React.ReactNode;
    className?: string;
}

export function InlineError({ children, className }: InlineErrorProps) {
    return (
        <div className={className ? `pure-inline-error ${className}` : 'pure-inline-error'}>
            {children}
        </div>
    );
}
