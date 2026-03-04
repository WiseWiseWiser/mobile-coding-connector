import './ButtonGroup.css';

export interface ButtonGroupProps {
    children: React.ReactNode;
    className?: string;
}

export function ButtonGroup({ children, className }: ButtonGroupProps) {
    return (
        <div className={className ? `pure-btn-group ${className}` : 'pure-btn-group'}>
            {children}
        </div>
    );
}
