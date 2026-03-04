import './Loading.css';

export interface LoadingProps {
    children?: React.ReactNode;
    className?: string;
}

export function Loading({ children = 'Loading...', className }: LoadingProps) {
    return (
        <div className={className ? `pure-loading ${className}` : 'pure-loading'}>
            {children}
        </div>
    );
}
