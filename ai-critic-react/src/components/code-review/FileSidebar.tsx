import type { DiffFile, GitDiffResult } from './types';
import { FileSection } from './FileSection';

interface FileSidebarProps {
    diffResult: GitDiffResult | null;
    selectedFile: DiffFile | null;
    onSelectFile: (file: DiffFile) => void;
    loading: boolean;
    onStageFile?: (file: DiffFile) => void;
    onStageDir?: (dir: string) => void;
}

export function FileSidebar({ diffResult, selectedFile, onSelectFile, loading, onStageFile, onStageDir }: FileSidebarProps) {
    const hasFiles = diffResult && diffResult.files && diffResult.files.length > 0;

    return (
        <div style={{ 
            width: '280px', 
            borderRight: '1px solid #e5e5e5',
            overflow: 'auto',
            backgroundColor: '#f9fafb',
        }}>
            {!hasFiles ? (
                <div style={{ padding: '20px', color: '#6b7280', textAlign: 'center', fontSize: '13px' }}>
                    {loading ? 'Loading...' : 'No changes detected'}
                </div>
            ) : (
                <div>
                    {/* Staged files */}
                    {diffResult?.files.filter(f => f.isStaged).length > 0 && (
                        <FileSection
                            title="STAGED CHANGES"
                            count={diffResult.files.filter(f => f.isStaged).length}
                            files={diffResult.files.filter(f => f.isStaged)}
                            selectedFile={selectedFile}
                            onSelectFile={onSelectFile}
                        />
                    )}
                    {/* Unstaged files */}
                    {diffResult?.files.filter(f => !f.isStaged).length > 0 && (
                        <FileSection
                            title="CHANGES"
                            count={diffResult.files.filter(f => !f.isStaged).length}
                            files={diffResult.files.filter(f => !f.isStaged)}
                            selectedFile={selectedFile}
                            onSelectFile={onSelectFile}
                            showStageButton={true}
                            onStageFile={onStageFile}
                            onStageDir={onStageDir}
                        />
                    )}
                </div>
            )}
        </div>
    );
}
