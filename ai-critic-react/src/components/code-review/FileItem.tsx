import type { DiffFile } from './types';
import { getFileIcon, getStatusBadge } from './utils';

interface FileItemProps {
    file: DiffFile;
    isSelected: boolean;
    onClick: () => void;
    indent: number;
    showStageButton?: boolean;
    onStage?: (file: DiffFile) => void;
}

export function FileItem({ file, isSelected, onClick, indent, showStageButton, onStage }: FileItemProps) {
    const fileName = file.path.split('/').pop() || file.path;
    const fileIcon = getFileIcon(file.path);
    const statusBadge = getStatusBadge(file.status);

    const handleStageClick = (e: React.MouseEvent) => {
        e.stopPropagation();
        onStage?.(file);
    };

    return (
        <div
            onClick={onClick}
            style={{
                padding: '4px 12px',
                paddingLeft: `${indent}px`,
                cursor: 'pointer',
                backgroundColor: isSelected ? '#dbeafe' : 'transparent',
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
            }}
            onMouseEnter={(e) => {
                if (!isSelected) {
                    e.currentTarget.style.backgroundColor = '#f3f4f6';
                }
            }}
            onMouseLeave={(e) => {
                if (!isSelected) {
                    e.currentTarget.style.backgroundColor = 'transparent';
                }
            }}
        >
            {/* File type icon */}
            <div style={{
                width: '18px',
                height: '18px',
                borderRadius: '3px',
                backgroundColor: fileIcon.color,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '8px',
                fontWeight: 700,
                color: '#fff',
                flexShrink: 0,
            }}>
                {fileIcon.letter.slice(0, 2)}
            </div>

            {/* File name */}
            <span style={{ 
                fontSize: '13px', 
                color: '#1f2937',
                flex: 1,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                textAlign: 'left',
            }}>
                {fileName}
            </span>

            {/* Stage button - always show for unstaged files */}
            {showStageButton && (
                <button
                    onClick={handleStageClick}
                    style={{
                        width: '18px',
                        height: '18px',
                        borderRadius: '3px',
                        backgroundColor: '#10b981',
                        border: 'none',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '14px',
                        fontWeight: 600,
                        color: '#fff',
                        cursor: 'pointer',
                        flexShrink: 0,
                    }}
                    title="Stage file"
                >
                    +
                </button>
            )}

            {/* Status badge */}
            <div style={{
                width: '16px',
                height: '16px',
                borderRadius: '3px',
                backgroundColor: statusBadge.bg,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '10px',
                fontWeight: 600,
                color: statusBadge.color,
                flexShrink: 0,
            }}>
                {statusBadge.letter}
            </div>
        </div>
    );
}
