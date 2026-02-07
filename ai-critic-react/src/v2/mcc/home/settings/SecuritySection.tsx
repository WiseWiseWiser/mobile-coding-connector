import { useState, useEffect } from 'react';
import { fetchEncryptKeyStatus, generateEncryptKeys } from '../../../../api/encrypt';
import type { EncryptKeyStatus } from '../../../../api/encrypt';
import { fetchCredentials, addCredentialToken, generateCredential, type MaskedCredential } from '../../../../api/auth';
import './SecuritySection.css';

export function SecuritySection() {
    const [keyStatus, setKeyStatus] = useState<EncryptKeyStatus | null>(null);
    const [credentials, setCredentials] = useState<MaskedCredential[]>([]);
    const [loading, setLoading] = useState(true);
    const [generating, setGenerating] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [newToken, setNewToken] = useState('');
    const [addingToken, setAddingToken] = useState(false);
    const [tokenError, setTokenError] = useState<string | null>(null);

    const loadStatus = () => {
        setLoading(true);
        setError(null);
        Promise.all([fetchEncryptKeyStatus(), fetchCredentials()])
            .then(([ks, creds]) => {
                setKeyStatus(ks);
                setCredentials(creds);
                setLoading(false);
            })
            .catch(err => { setError(err.message); setLoading(false); });
    };

    useEffect(() => {
        loadStatus();
    }, []);

    const handleGenerate = async () => {
        setGenerating(true);
        setError(null);
        try {
            const status = await generateEncryptKeys();
            setKeyStatus(status);
        } catch (err) {
            setError(err instanceof Error ? err.message : String(err));
        }
        setGenerating(false);
    };

    const handleAddToken = async () => {
        const token = newToken.trim();
        if (!token) return;
        setAddingToken(true);
        setTokenError(null);
        try {
            await addCredentialToken(token);
            setNewToken('');
            // Refresh credentials list
            const creds = await fetchCredentials();
            setCredentials(creds);
        } catch (err) {
            setTokenError(err instanceof Error ? err.message : String(err));
        }
        setAddingToken(false);
    };

    const handleGenerateRandom = async () => {
        setTokenError(null);
        try {
            const token = await generateCredential();
            setNewToken(token);
        } catch (err) {
            setTokenError(err instanceof Error ? err.message : String(err));
        }
    };

    return (
        <div className="diagnose-section">
            <h3 className="diagnose-section-title">Security</h3>

            {loading ? (
                <div className="diagnose-loading">Checking security status...</div>
            ) : error ? (
                <div className="diagnose-error">{error}</div>
            ) : (
                <>
                    {/* Credentials */}
                    <div className="diagnose-security-card">
                        <div className="diagnose-security-header">
                            <span className="diagnose-security-status">
                                {credentials.length > 0 ? '\u2705' : '\u274C'}
                            </span>
                            <div className="diagnose-security-info">
                                <span className="diagnose-security-label">Server Credentials</span>
                                <span className="diagnose-security-desc">
                                    {credentials.length > 0
                                        ? `${credentials.length} token(s) configured for server authentication.`
                                        : 'No credentials configured. Server authentication may not work.'}
                                </span>
                            </div>
                        </div>
                        {credentials.length > 0 && (
                            <div className="diagnose-security-credentials">
                                {credentials.map((c, i) => (
                                    <div key={i} className="diagnose-security-credential-row">
                                        <code className="diagnose-security-credential-value">{c.masked}</code>
                                    </div>
                                ))}
                            </div>
                        )}
                        <div className="diagnose-security-add-token">
                            <input
                                type="text"
                                className="diagnose-security-token-input"
                                placeholder="Enter token or generate one..."
                                value={newToken}
                                onChange={e => setNewToken(e.target.value)}
                                disabled={addingToken}
                            />
                            <div className="diagnose-security-token-actions">
                                <button
                                    className="diagnose-security-btn"
                                    onClick={handleAddToken}
                                    disabled={addingToken || !newToken.trim()}
                                >
                                    {addingToken ? 'Adding...' : 'Add Token'}
                                </button>
                                <button
                                    className="diagnose-security-btn diagnose-security-btn--secondary"
                                    onClick={handleGenerateRandom}
                                    disabled={addingToken}
                                >
                                    Generate Random
                                </button>
                            </div>
                            {tokenError && (
                                <div className="diagnose-security-error">{tokenError}</div>
                            )}
                        </div>
                    </div>

                    {/* Encryption Keys */}
                    {keyStatus && (
                        <div className="diagnose-security-card">
                            <div className="diagnose-security-header">
                                <span className="diagnose-security-status">
                                    {keyStatus.valid ? '\u2705' : keyStatus.exists ? '\u26A0\uFE0F' : '\u274C'}
                                </span>
                                <div className="diagnose-security-info">
                                    <span className="diagnose-security-label">Encryption Key Pair</span>
                                    <span className="diagnose-security-desc">
                                        {keyStatus.valid
                                            ? 'RSA key pair is valid and available for encrypting sensitive data in transit.'
                                            : keyStatus.exists
                                                ? 'Key files found but validation failed. Regenerate to fix.'
                                                : 'No encryption key pair found. Generate one to enable secure data transfer.'}
                                    </span>
                                </div>
                            </div>

                            {keyStatus.error && (
                                <div className="diagnose-security-error">{keyStatus.error}</div>
                            )}

                            <div className="diagnose-security-paths">
                                <div className="diagnose-security-path-row">
                                    <span className="diagnose-security-path-label">Private key:</span>
                                    <code className="diagnose-security-path-value">{keyStatus.private_key_path}</code>
                                </div>
                                <div className="diagnose-security-path-row">
                                    <span className="diagnose-security-path-label">Public key:</span>
                                    <code className="diagnose-security-path-value">{keyStatus.public_key_path}</code>
                                </div>
                            </div>

                            <button
                                className="diagnose-security-btn"
                                onClick={handleGenerate}
                                disabled={generating}
                            >
                                {generating
                                    ? 'Generating...'
                                    : keyStatus.valid
                                        ? 'Regenerate Key Pair'
                                        : keyStatus.exists
                                            ? 'Fix: Regenerate Key Pair'
                                            : 'Generate Key Pair'}
                            </button>
                        </div>
                    )}
                </>
            )}
        </div>
    );
}
