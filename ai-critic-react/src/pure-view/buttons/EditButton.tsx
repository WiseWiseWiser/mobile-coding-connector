import { EditIcon } from '../icons/EditIcon';
import './EditButton.css';

export interface EditButtonProps {
    onClick: (e: React.MouseEvent) => void;
    title?: string;
    className?: string;
}

export function EditButton({ onClick, title = 'Edit', className }: EditButtonProps) {
    return (
        <button
            className={className ? `pure-edit-btn ${className}` : 'pure-edit-btn'}
            onClick={onClick}
            title={title}
        >
            <EditIcon />
        </button>
    );
}
