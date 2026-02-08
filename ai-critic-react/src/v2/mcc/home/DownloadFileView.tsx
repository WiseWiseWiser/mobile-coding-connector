import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { downloadFile } from '../../../api/filedownload';
import { ServerFileBrowser } from './ServerFileBrowser';
import { TransferProgress } from './TransferProgress';
import type { TransferProgressData } from './TransferProgress';
import './DownloadFileView.css';

export function DownloadFileView() {
    const navigate = useNavigate();
    const [selectedFile, setSelectedFile] = useState<string | null>(null);
    const [downloading, setDownloading] = useState(false);
    const [downloadProgress, setDownloadProgress] = useState<TransferProgressData | null>(null);
    const [error, setError] = useState<string | null>(null);

    const handleDownload = async () => {
        if (!selectedFile) return;
        setDownloading(true);
        setDownloadProgress(null);
        setError(null);
        try {
            await downloadFile(selectedFile, setDownloadProgress);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setDownloading(false);
    };

    return (
        <div className="download-view">
            <div className="mcc-section-header">
                <button className="mcc-back-btn" onClick={() => navigate('..')}>&larr;</button>
                <h2>Download File</h2>
            </div>

            <div className="download-form">
                <ServerFileBrowser
                    selectMode="file"
                    onSelect={setSelectedFile}
                />

                {/* Error */}
                {error && <div className="download-error">{error}</div>}

                {/* Download Progress */}
                {downloading && <TransferProgress progress={downloadProgress} label="Download" />}

                {/* Download Button */}
                <button
                    className="download-submit-btn"
                    onClick={handleDownload}
                    disabled={!selectedFile || downloading}
                >
                    {downloading
                        ? 'Downloading...'
                        : selectedFile
                            ? `Download ${selectedFile.split('/').pop()}`
                            : 'Select a file to download'}
                </button>
            </div>
        </div>
    );
}
