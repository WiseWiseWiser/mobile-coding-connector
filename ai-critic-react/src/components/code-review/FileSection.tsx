import { useState } from 'react';
import type { DiffFile } from './types';
import { groupFilesByDirectory } from './utils';
import { FileItem } from './FileItem';
import { FolderIcon } from './FolderIcon';

interface FileSectionProps {
    title: string;
    count: number;
    files: DiffFile[];
    selectedFile: DiffFile | null;
    onSelectFile: (file: DiffFile) => void;
    showStageButton?: boolean;
    onStageFile?: (file: DiffFile) => void;
    onStageDir?: (dir: string) => void;
}

export function FileSection({ title, count, files, selectedFile, onSelectFile, showStageButton, onStageFile, onStageDir }: FileSectionProps) {
    const [isExpanded, setIsExpanded] = useState(true);
    const grouped = groupFilesByDirectory(files);

    return (
        <div style={{ borderBottom: '1px solid #e5e7eb' }}>
            {/* Section header */}
            <div 
                onClick={() => setIsExpanded(!isExpanded)}
                style={{ 
                    padding: '8px 12px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px',
                    cursor: 'pointer',
                    backgroundColor: '#f3f4f6',
                    userSelect: 'none',
                }}
            >
                <svg 
                    width="12" 
                    height="12" 
                    viewBox="0 0 12 12"
                    style={{ 
                        transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                        transition: 'transform 0.15s ease',
                    }}
                >
                    <path d="M4 2 L8 6 L4 10" fill="none" stroke="#6b7280" strokeWidth="1.5" strokeLinecap="round" />
                </svg>
                <span style={{ 
                    fontSize: '11px', 
                    fontWeight: 600, 
                    color: '#374151',
                    letterSpacing: '0.5px',
                }}>
                    {title}
                </span>
                <span style={{ 
                    fontSize: '11px', 
                    color: '#6b7280',
                    backgroundColor: '#e5e7eb',
                    padding: '1px 6px',
                    borderRadius: '10px',
                    marginLeft: 'auto',
                }}>
                    {count}
                </span>
            </div>

            {/* File list */}
            {isExpanded && (
                <div>
                    {Array.from(grouped.entries()).map(([dir, dirFiles]) => (
                        <div key={dir || 'root'}>
                            {dir && (
                                <div style={{ 
                                    padding: '4px 12px 4px 20px',
                                    fontSize: '11px',
                                    color: '#6b7280',
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '4px',
                                }}>
                                    <FolderIcon />
                                    <span style={{
                                        flex: 1,
                                        whiteSpace: 'nowrap',
                                        overflow: 'hidden',
                                        textOverflow: 'ellipsis',
                                    }}>{dir}</span>
                                    {showStageButton && onStageDir && (
                                        <button
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                onStageDir(dir);
                                            }}
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
                                            title="Stage directory"
                                        >
                                            +
                                        </button>
                                    )}
                                </div>
                            )}
                            {dirFiles.map((file, idx) => (
                                <FileItem 
                                    key={idx}
                                    file={file}
                                    isSelected={selectedFile === file}
                                    onClick={() => onSelectFile(file)}
                                    indent={dir ? 32 : 20}
                                    showStageButton={showStageButton}
                                    onStage={onStageFile}
                                />
                            ))}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
