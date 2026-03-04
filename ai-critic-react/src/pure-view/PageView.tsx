import './PageView.css';

export interface PageViewProps {
    children: React.ReactNode;
    className?: string;
}

export function PageView({ children, className }: PageViewProps) {
    return (
        <div className={className ? `pure-page-view ${className}` : 'pure-page-view'}>
            {children}
        </div>
    );
}
