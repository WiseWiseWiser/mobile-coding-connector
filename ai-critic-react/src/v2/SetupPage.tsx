import { useState } from 'react';
import { setupCredential, generateCredential } from '../api/auth';
import './SetupPage.css';

interface SetupPageProps {
    onSetupComplete: () => void;
}

export function SetupPage({ onSetupComplete }: SetupPageProps) {
    const [credential, setCredential] = useState('');
    const [copied, setCopied] = useState(false);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [generating, setGenerating] = useState(false);

    const handleGenerate = async () => {
        setGenerating(true);
        setError('');
        try {
            const token = await generateCredential();
            setCredential(token);
            setCopied(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setGenerating(false);
    };

    const handleCopy = async () => {
        try {
            await navigator.clipboard.writeText(credential);
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        } catch {
            setError('Failed to copy to clipboard');
        }
    };

    const handleConfirm = async () => {
        if (!credential.trim()) {
            setError('Please enter or generate a credential first');
            return;
        }

        setLoading(true);
        setError('');

        try {
            const resp = await setupCredential(credential.trim());
            const data = await resp.json();
            if (!resp.ok) {
                setError(data.error || 'Setup failed');
                setLoading(false);
                return;
            }
            onSetupComplete();
        } catch (err) {
            setError(String(err));
            setLoading(false);
        }
    };

    return (
        <div className="mcc-setup">
            <div className="mcc-setup-card">
                <h1 className="mcc-setup-title">AI Critic</h1>
                <p className="mcc-setup-subtitle">Server is not initialized yet. Set up an initial credential to secure your server.</p>

                <div className="mcc-setup-actions">
                    <div className="mcc-setup-credential">
                        <label className="mcc-setup-label">Credential</label>
                        <input
                            type="text"
                            className="mcc-setup-credential-input"
                            placeholder="Enter a credential or click Generate..."
                            value={credential}
                            onChange={e => { setCredential(e.target.value); setCopied(false); }}
                        />
                        <div className="mcc-setup-credential-btns">
                            <button
                                type="button"
                                className="mcc-setup-btn mcc-setup-btn-generate"
                                onClick={handleGenerate}
                                disabled={generating}
                            >
                                {generating ? 'Generating...' : 'Generate Random'}
                            </button>
                            {credential && (
                                <button
                                    type="button"
                                    className="mcc-setup-btn mcc-setup-btn-copy"
                                    onClick={handleCopy}
                                >
                                    {copied ? 'Copied!' : 'Copy'}
                                </button>
                            )}
                        </div>
                    </div>

                    {error && <div className="mcc-setup-error">{error}</div>}

                    {credential && (
                        <div className="mcc-setup-note">
                            Save this credential before confirming. You will need it to log in.
                        </div>
                    )}

                    <button
                        type="button"
                        className="mcc-setup-btn mcc-setup-btn-confirm"
                        onClick={handleConfirm}
                        disabled={loading || !credential.trim()}
                    >
                        {loading ? 'Setting up...' : 'Confirm & Continue'}
                    </button>
                </div>
            </div>
        </div>
    );
}
