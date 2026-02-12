import './FileActionSheet.css';

interface FileInfo {
    name: string;
    path: string;
    size: number;
    isDir: boolean;
    modifiedTime?: string;
}

interface FileActionSheetProps {
    file: FileInfo | null;
    onClose: () => void;
    onView: () => void;
    onEdit: () => void;
}

function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function formatDate(dateStr?: string): string {
    if (!dateStr) return 'Unknown';
    const date = new Date(dateStr);
    return date.toLocaleString();
}

export function FileActionSheet({ file, onClose, onView, onEdit }: FileActionSheetProps) {
    console.log('FileActionSheet rendered, file:', file);
    
    const handleClose = () => {
        onClose();
    };

    if (!file) {
        console.log('FileActionSheet: no file, returning null');
        return null;
    }

    return (
        <>
            <div className="file-action-sheet-overlay" onClick={handleClose} />
            <div className="file-action-sheet">
                <div className="file-action-sheet-handle" />
                <div className="file-action-sheet-content">
                    <div className="file-action-sheet-header">
                        <div className="file-action-sheet-icon">📄</div>
                        <div className="file-action-sheet-name">{file.name}</div>
                    </div>

                    <div className="file-action-sheet-info">
                        <div className="file-action-sheet-info-row">
                            <span className="file-action-sheet-info-label">Size</span>
                            <span className="file-action-sheet-info-value">{formatFileSize(file.size)}</span>
                        </div>
                        <div className="file-action-sheet-info-row">
                            <span className="file-action-sheet-info-label">Path</span>
                            <span className="file-action-sheet-info-value">{file.path}</span>
                        </div>
                        {file.modifiedTime && (
                            <div className="file-action-sheet-info-row">
                                <span className="file-action-sheet-info-label">Modified</span>
                                <span className="file-action-sheet-info-value">{formatDate(file.modifiedTime)}</span>
                            </div>
                        )}
                    </div>

                    <div className="file-action-sheet-actions">
                        <button className="file-action-sheet-btn view" onClick={() => { onView(); handleClose(); }}>
                            👁 View
                        </button>
                        <button className="file-action-sheet-btn edit" onClick={() => { onEdit(); handleClose(); }}>
                            ✎ Edit
                        </button>
                        <button className="file-action-sheet-btn cancel" onClick={handleClose}>
                            Cancel
                        </button>
                    </div>
                </div>
            </div>
        </>
    );
}
