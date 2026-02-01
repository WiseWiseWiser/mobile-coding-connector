import { useState, useEffect } from 'react';
import type { 
    DiffFile, 
    GitDiffResult, 
    ProviderInfo,
    ModelInfo,
} from './components/code-review/types';
import { Header } from './components/code-review/Header';
import { FileSidebar } from './components/code-review/FileSidebar';
import { DiffViewer } from './components/code-review/DiffViewer';
import { ChatPanel } from './components/code-review/ChatPanel';
import * as reviewApi from './api/review';

// Diff mode for AI review:
// - "unstaged": only send unstaged changes (including untracked files) to AI
// - "all": send both staged and unstaged changes to AI
export const DiffModes = {
    Unstaged: "unstaged",
    All: "all",
} as const;

export type DiffMode = typeof DiffModes[keyof typeof DiffModes];

// Global variable to control which diff to send to AI
// Change this to test different modes
export let DIFF_MODE: DiffMode = DiffModes.Unstaged;

// Build diff context with file line info for AI
function buildDiffContext(diffResult: GitDiffResult): string {
    // Filter files based on mode
    const filesToInclude = DIFF_MODE === DiffModes.Unstaged
        ? diffResult.files.filter(f => !f.isStaged)
        : diffResult.files;
    
    // Build file info summary
    const fileInfoLines = filesToInclude.map(f => {
        const lines = f.totalLines > 0 ? `${f.totalLines} lines` : 'deleted';
        return `- ${f.path}: ${f.status} (${lines})`;
    });
    
    const fileInfo = fileInfoLines.length > 0 
        ? `File Information:\n${fileInfoLines.join('\n')}\n\n` 
        : '';
    
    // Build diff content based on mode
    if (DIFF_MODE === DiffModes.Unstaged) {
        // Only unstaged changes (including untracked)
        return fileInfo + diffResult.workingTreeDiff;
    }
    // All changes: both staged and unstaged
    return fileInfo + diffResult.workingTreeDiff + '\n' + diffResult.stagedDiff;
}

function CodeReview() {
    const [dir, setDir] = useState('');
    const [loading, setLoading] = useState(false);
    const [diffResult, setDiffResult] = useState<GitDiffResult | null>(null);
    const [selectedFile, setSelectedFile] = useState<DiffFile | null>(null);
    const [error, setError] = useState<string | null>(null);
    
    // Provider/Model state
    const [providers, setProviders] = useState<ProviderInfo[]>([]);
    const [models, setModels] = useState<ModelInfo[]>([]);
    const [selectedProvider, setSelectedProvider] = useState('');
    const [selectedModel, setSelectedModel] = useState('');

    // Fetch initial config and load diff on mount
    useEffect(() => {
        const init = async () => {
            try {
                // Get initial config
                const config = await reviewApi.getConfig();
                
                // Set providers and models
                if (config.providers) {
                    setProviders(config.providers);
                }
                if (config.models) {
                    setModels(config.models);
                }
                
                // Set default provider/model
                if (config.defaultProvider) {
                    setSelectedProvider(config.defaultProvider);
                } else if (config.providers && config.providers.length > 0) {
                    setSelectedProvider(config.providers[0].name);
                }
                if (config.defaultModel) {
                    setSelectedModel(config.defaultModel);
                } else if (config.models && config.models.length > 0) {
                    setSelectedModel(config.models[0].model);
                }
                
                if (config.initialDir) {
                    setDir(config.initialDir);
                    
                    // Auto-load diff with the initial directory
                    setLoading(true);
                    const diffData = await reviewApi.getDiff(config.initialDir);
                    if (!diffData.error) {
                        setDiffResult(diffData);
                        // Select first file if available
                        if (diffData.files && diffData.files.length > 0) {
                            setSelectedFile(diffData.files[0]);
                        }
                    }
                }
            } catch (err) {
                console.error('Failed to load initial config:', err);
            } finally {
                setLoading(false);
            }
        };
        init();
    }, []);

    const handleGetDiff = async () => {
        setLoading(true);
        setError(null);
        // Don't clear diffResult and selectedFile to avoid UI fluttering during reload

        try {
            const data = await reviewApi.getDiff(dir || undefined);
            if (data.error) {
                setError(data.error);
                setDiffResult(null);
                setSelectedFile(null);
            } else {
                setDiffResult(data);
                // Only select first file if no file is currently selected or if the selected file no longer exists
                if (!selectedFile || !data.files?.some(f => f.path === selectedFile.path)) {
                    if (data.files && data.files.length > 0) {
                        setSelectedFile(data.files[0]);
                    } else {
                        setSelectedFile(null);
                    }
                }
            }
        } catch (err) {
            setError(`Failed to get diff: ${err}`);
        } finally {
            setLoading(false);
        }
    };

    const handleStageFile = async (file: DiffFile) => {
        try {
            await reviewApi.stageFile(file.path, dir || undefined);
            // Refresh the diff to show updated staging status
            handleGetDiff();
        } catch (err) {
            setError(`Failed to stage file: ${err}`);
        }
    };

    const handleStageDir = async (dirPath: string) => {
        try {
            await reviewApi.stageFile(dirPath, dir || undefined);
            // Refresh the diff to show updated staging status
            handleGetDiff();
        } catch (err) {
            setError(`Failed to stage directory: ${err}`);
        }
    };

    const hasFiles = diffResult && diffResult.files && diffResult.files.length > 0;

    return (
        <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
            <Header 
                dir={dir}
                onDirChange={setDir}
                loading={loading}
                onRefresh={handleGetDiff}
            />

            {error && (
                <div style={{
                    padding: '12px 20px',
                    backgroundColor: '#fef2f2',
                    borderBottom: '1px solid #fecaca',
                    color: '#dc2626',
                    fontSize: '13px',
                }}>
                    <strong>Error:</strong> {error}
                </div>
            )}

            {/* Main content */}
            <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
                <FileSidebar 
                    diffResult={diffResult}
                    selectedFile={selectedFile}
                    onSelectFile={setSelectedFile}
                    loading={loading}
                    onStageFile={handleStageFile}
                    onStageDir={handleStageDir}
                />

                {/* Diff viewer */}
                <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
                    <DiffViewer selectedFile={selectedFile} />
                </div>

                {/* Chat panel on right with model selector and review */}
                <ChatPanel 
                    diffContext={diffResult ? buildDiffContext(diffResult) : ''}
                    provider={selectedProvider}
                    model={selectedModel}
                    providers={providers}
                    models={models}
                    onProviderChange={setSelectedProvider}
                    onModelChange={setSelectedModel}
                    loading={loading}
                    hasFiles={!!hasFiles}
                />
            </div>
        </div>
    );
}

export default CodeReview;
