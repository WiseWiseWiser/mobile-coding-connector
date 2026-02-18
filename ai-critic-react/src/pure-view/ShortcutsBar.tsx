import './ShortcutsBar.css';

export interface ShortcutsBarProps {
    onSendKey: (key: string) => void;
    className?: string;
}

export function ShortcutsBar({ onSendKey, className = '' }: ShortcutsBarProps) {
    return (
        <div className={`shortcuts-bar ${className}`}>
            <button className="shortcuts-bar-btn" onClick={() => onSendKey('\t')}>Tab</button>
            <button className="shortcuts-bar-btn" onClick={() => onSendKey('\x1b[A')}>↑</button>
            <button className="shortcuts-bar-btn" onClick={() => onSendKey('\x1b[B')}>↓</button>
            <button className="shortcuts-bar-btn" onClick={() => onSendKey('\x03')}>^C</button>
            <button className="shortcuts-bar-btn" onClick={() => onSendKey('\x0c')}>^L</button>
        </div>
    );
}
