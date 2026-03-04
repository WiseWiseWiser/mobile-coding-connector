import type { ReactNode } from 'react';
import './ModalOverlay.css';

interface ModalOverlayProps {
    children: ReactNode;
    onClose: () => void;
}

export function ModalOverlay({ children, onClose }: ModalOverlayProps) {
    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal-container" onClick={e => e.stopPropagation()}>
                {children}
            </div>
        </div>
    );
}
