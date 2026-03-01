import { UploadIcon } from '../icons/UploadIcon';
import './button-common.css';
import './ImportButton.css';

interface ImportButtonProps {
    onClick: () => void;
    className?: string;
}

export function ImportButton({ onClick, className }: ImportButtonProps) {
    return (
        <button
            className={`btn-base import-btn${className ? ` ${className}` : ''}`}
            onClick={onClick}
        >
            <UploadIcon className="btn-icon" />
            Import
        </button>
    );
}
