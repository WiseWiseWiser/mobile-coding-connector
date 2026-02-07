import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Terminal } from '../terminal/Terminal';
import './TopNavBar.css';

interface NavItem {
    path: string;
    label: string;
}

const navItems: NavItem[] = [
    { path: '/v1', label: 'Code Review' },
    { path: '/v1/home', label: 'Home' },
    { path: '/v1/gen', label: 'Generator' },
    // Mockups page only in dev mode
    ...(import.meta.env.DEV ? [{ path: '/v1/mockups', label: 'Mockups' }] : []),
];

export function TopNavBar() {
    const [isMenuOpen, setIsMenuOpen] = useState(false);
    const [isTerminalOpen, setIsTerminalOpen] = useState(false);
    const location = useLocation();

    const toggleMenu = () => {
        setIsMenuOpen(!isMenuOpen);
    };

    const closeMenu = () => {
        setIsMenuOpen(false);
    };

    const toggleTerminal = () => {
        setIsTerminalOpen(!isTerminalOpen);
        closeMenu();
    };

    return (
        <>
            <nav className="top-nav">
                <div className="nav-brand">
                    <Link to="/v1" className="brand-link">
                        <img src="/ai-critic.svg" alt="AI Critic" className="brand-logo" />
                        <span className="brand-text">AI Critic</span>
                    </Link>
                </div>

                {/* Desktop Navigation */}
                <div className="nav-links desktop-only">
                    {navItems.map(item => (
                        <Link
                            key={item.path}
                            to={item.path}
                            className={`nav-link ${location.pathname === item.path ? 'active' : ''}`}
                        >
                            {item.label}
                        </Link>
                    ))}
                </div>

                <div className="nav-actions">
                    <button 
                        className="terminal-btn"
                        onClick={toggleTerminal}
                        title="Open Terminal"
                    >
                        <TerminalIcon />
                        <span className="btn-text">Terminal</span>
                    </button>

                    {/* Mobile Menu Button */}
                    <button 
                        className="menu-btn mobile-only"
                        onClick={toggleMenu}
                        aria-label="Toggle menu"
                    >
                        {isMenuOpen ? <CloseIcon /> : <MenuIcon />}
                    </button>
                </div>

                {/* Mobile Navigation Dropdown */}
                {isMenuOpen && (
                    <div className="mobile-menu">
                        {navItems.map(item => (
                            <Link
                                key={item.path}
                                to={item.path}
                                className={`mobile-nav-link ${location.pathname === item.path ? 'active' : ''}`}
                                onClick={closeMenu}
                            >
                                {item.label}
                            </Link>
                        ))}
                    </div>
                )}
            </nav>

            <Terminal isOpen={isTerminalOpen} onClose={() => setIsTerminalOpen(false)} />
        </>
    );
}

function TerminalIcon() {
    return (
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="4 17 10 11 4 5"></polyline>
            <line x1="12" y1="19" x2="20" y2="19"></line>
        </svg>
    );
}

function MenuIcon() {
    return (
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <line x1="3" y1="12" x2="21" y2="12"></line>
            <line x1="3" y1="6" x2="21" y2="6"></line>
            <line x1="3" y1="18" x2="21" y2="18"></line>
        </svg>
    );
}

function CloseIcon() {
    return (
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
    );
}
