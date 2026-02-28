import './ExternalAgentLink.css';

export interface ExternalAgentLinkProps {
    href: string;
    children: React.ReactNode;
    className?: string;
}

export function ExternalAgentLink({ href, children, className }: ExternalAgentLinkProps) {
    const classes = className ? `external-agent-link ${className}` : 'external-agent-link';
    return (
        <a className={classes} href={href} target="_blank" rel="noreferrer">
            {children}
        </a>
    );
}
