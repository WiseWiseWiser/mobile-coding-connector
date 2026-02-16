import type { ReactNode } from 'react';
import './MockupPageContainer.css';

interface MockupPageContainerProps {
    title: string;
    description?: string;
    children: ReactNode;
}

export function MockupPageContainer({ title, description, children }: MockupPageContainerProps) {
    return (
        <div className="mockup-page-container">
            <div className="mockup-page-header">
                <h2 className="mockup-page-title">{title}</h2>
                {description && (
                    <p className="mockup-page-description">{description}</p>
                )}
            </div>
            <div className="mockup-page-content">
                {children}
            </div>
        </div>
    );
}
