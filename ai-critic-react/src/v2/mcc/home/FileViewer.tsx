import { useState, useEffect } from 'react';
import { fetchFilePartial } from '../../../api/files';
import './FileActionSheet.css';

interface FileViewerProps {
    filePath: string;
    onClose: () => void;
}

const PAGE_SIZE = 8192; // 8KB per page

function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function FileViewer({ filePath, onClose }: FileViewerProps) {
    const [content, setContent] = useState('');
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [currentPage, setCurrentPage] = useState(0);
    const [totalSize, setTotalSize] = useState(0);
    const [totalPages, setTotalPages] = useState(1);

    useEffect(() => {
        loadPage(0);
    }, [filePath]);

    const loadPage = async (page: number) => {
        setLoading(true);
        setError('');
        try {
            const result = await fetchFilePartial(filePath, page * PAGE_SIZE, PAGE_SIZE);
            setContent(result.content);
            setTotalSize(result.totalSize);
            setTotalPages(Math.ceil(result.totalSize / PAGE_SIZE));
            setCurrentPage(page);
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to load file');
        } finally {
            setLoading(false);
        }
    };

    const handlePrevPage = () => {
        if (currentPage > 0) {
            loadPage(currentPage - 1);
        }
    };

    const handleNextPage = () => {
        if (currentPage < totalPages - 1) {
            loadPage(currentPage + 1);
        }
    };

    return (
        <div className="file-viewer-overlay" onClick={onClose}>
            <div className="file-viewer" onClick={(e) => e.stopPropagation()}>
                <div className="file-viewer-header">
                    <span className="file-viewer-title">
                        {filePath.split('/').pop()}
                        {totalSize > 0 && (
                            <span style={{ marginLeft: 12, fontSize: 12, color: '#94a3b8', fontWeight: 400 }}>
                                ({formatBytes(totalSize)})
                            </span>
                        )}
                    </span>
                    <button className="file-viewer-close" onClick={onClose}>×</button>
                </div>
                
                <div className="file-viewer-content">
                    {loading ? (
                        <div style={{ textAlign: 'center', padding: 40, color: '#94a3b8' }}>Loading...</div>
                    ) : error ? (
                        <div style={{ textAlign: 'center', padding: 40, color: '#ef4444' }}>{error}</div>
                    ) : (
                        <pre className="file-viewer-text">{content}</pre>
                    )}
                </div>

                {totalPages > 1 && (
                    <div className="file-viewer-pagination">
                        <button 
                            className="file-viewer-page-btn" 
                            onClick={handlePrevPage}
                            disabled={currentPage === 0 || loading}
                        >
                            ← Previous
                        </button>
                        <span className="file-viewer-page-info">
                            Page {currentPage + 1} of {totalPages}
                        </span>
                        <button 
                            className="file-viewer-page-btn" 
                            onClick={handleNextPage}
                            disabled={currentPage >= totalPages - 1 || loading}
                        >
                            Next →
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
}
