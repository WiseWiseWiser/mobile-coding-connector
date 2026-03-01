import { useState, useEffect } from 'react';
import { gitPushStream, getGitBranches } from '../../../api/review';
import type { GitBranch } from '../../../api/review';
import { encryptProjectSSHKey, EncryptionNotAvailableError } from '../home/crypto';
import { useStreamingAction } from '../../../hooks/useStreamingAction';
import { StreamingLogs } from '../../StreamingComponents';
import { SSHKeyRequiredHint } from '../components/SSHKeyRequiredHint';
import { UploadIcon } from '../../../pure-view/icons/UploadIcon';
import './GitCommitView.css';

export interface GitPushSectionProps {
    projectDir: string;
    sshKeyId?: string;
}

export function GitPushSection({ projectDir, sshKeyId }: GitPushSectionProps) {
    const [branches, setBranches] = useState<GitBranch[]>([]);
    const [pushBranch, setPushBranch] = useState('');
    const [encryptionError, setEncryptionError] = useState<string | null>(null);
    const [pushState, pushControls] = useStreamingAction();

    useEffect(() => {
        getGitBranches(projectDir)
            .then(branchList => {
                setBranches(branchList || []);
                const current = branchList?.find(b => b.isCurrent);
                if (current && !pushBranch) {
                    setPushBranch(current.name);
                }
            })
            .catch(() => setBranches([]));
    }, [projectDir]); // eslint-disable-line react-hooks/exhaustive-deps

    const handlePush = () => {
        pushControls.run(async () => {
            setEncryptionError(null);
            let encryptedKey: string | undefined;
            try {
                encryptedKey = await encryptProjectSSHKey(sshKeyId);
            } catch (err) {
                if (err instanceof EncryptionNotAvailableError) {
                    setEncryptionError('Server encryption keys not configured.');
                    throw err;
                }
                throw err;
            }
            return gitPushStream(projectDir, encryptedKey);
        });
    };

    const hasSSHKey = !!sshKeyId;

    return (
        <div>
            {!hasSSHKey && (
                <SSHKeyRequiredHint message="SSH key required for push operations. Configure in project settings." style={{ marginBottom: 10 }} />
            )}
            <div className="mcc-git-push-row">
                <button
                    className="mcc-git-push-btn"
                    onClick={handlePush}
                    disabled={pushState.running || !hasSSHKey}
                    style={{ opacity: !hasSSHKey ? 0.5 : 1 }}
                >
                    {pushState.running ? 'Pushing...' : 'Push'}
                    <UploadIcon size={14} style={{ verticalAlign: 'middle' }} />
                </button>
                {branches.length > 0 && (
                    <select
                        className="mcc-git-branch-select"
                        value={pushBranch}
                        onChange={(e) => setPushBranch(e.target.value)}
                    >
                        {branches.map(b => (
                            <option key={b.name} value={b.name}>
                                {b.name}{b.isCurrent ? ' (current)' : ''}
                            </option>
                        ))}
                    </select>
                )}
            </div>
            <StreamingLogs
                state={pushState}
                pendingMessage="Pushing..."
                maxHeight={200}
            />
            {encryptionError && (
                <div className="mcc-git-fetch-result error" style={{ marginTop: 8 }}>{encryptionError}</div>
            )}
        </div>
    );
}
