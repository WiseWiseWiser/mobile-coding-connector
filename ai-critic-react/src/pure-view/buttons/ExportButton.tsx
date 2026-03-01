import { DownloadIcon } from '../icons/DownloadIcon';
import './button-common.css';
import './ExportButton.css';

interface ExportButtonProps {
    onClick: () => void;
    className?: string;
}

export function ExportButton({ onClick, className }: ExportButtonProps) {
    return (
        <button
            className={`btn-base export-btn${className ? ` ${className}` : ''}`}
            onClick={onClick}
        >
            <DownloadIcon className="btn-icon" />
            Export
        </button>
    );
}
