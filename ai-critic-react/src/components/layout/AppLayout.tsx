import type { ReactNode } from 'react';
import { TopNavBar } from './TopNavBar';
import './AppLayout.css';

interface AppLayoutProps {
    children: ReactNode;
    hideNav?: boolean;
}

export function AppLayout({ children, hideNav = false }: AppLayoutProps) {
    return (
        <div className="app-layout">
            {!hideNav && <TopNavBar />}
            <main className="app-main">
                {children}
            </main>
        </div>
    );
}
