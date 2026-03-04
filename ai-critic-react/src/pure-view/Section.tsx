import './Section.css';

export interface SectionProps {
    title: string;
    children: React.ReactNode;
    className?: string;
}

export function Section({ title, children, className }: SectionProps) {
    return (
        <div className={className ? `pure-section ${className}` : 'pure-section'}>
            <h3 className="pure-section-title">{title}</h3>
            {children}
        </div>
    );
}
