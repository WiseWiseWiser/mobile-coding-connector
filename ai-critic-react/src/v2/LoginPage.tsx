import { useState } from 'react';
import { login } from '../api/auth';
import './LoginPage.css';

interface LoginPageProps {
    onLoginSuccess: () => void;
}

export function LoginPage({ onLoginSuccess }: LoginPageProps) {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!username.trim() || !password.trim()) {
            setError('Username and password are required');
            return;
        }

        setLoading(true);
        setError('');

        try {
            const resp = await login(username.trim(), password.trim());
            const data = await resp.json();

            if (!resp.ok) {
                setError(data.error || 'Login failed');
                setLoading(false);
                return;
            }

            onLoginSuccess();
        } catch (err) {
            setError(String(err));
            setLoading(false);
        }
    };

    return (
        <div className="mcc-login">
            <div className="mcc-login-card">
                <h1 className="mcc-login-title">AI Critic</h1>
                <p className="mcc-login-subtitle">Sign in to continue</p>
                <form onSubmit={handleSubmit} className="mcc-login-form">
                    <div className="mcc-login-field">
                        <label htmlFor="username">Username</label>
                        <input
                            id="username"
                            type="text"
                            placeholder="Enter username"
                            value={username}
                            onChange={e => setUsername(e.target.value)}
                            autoComplete="username"
                            autoFocus
                        />
                    </div>
                    <div className="mcc-login-field">
                        <label htmlFor="password">Password</label>
                        <input
                            id="password"
                            type="password"
                            placeholder="Enter password"
                            value={password}
                            onChange={e => setPassword(e.target.value)}
                            autoComplete="current-password"
                        />
                    </div>
                    {error && <div className="mcc-login-error">{error}</div>}
                    <button type="submit" className="mcc-login-btn" disabled={loading}>
                        {loading ? 'Signing in...' : 'Sign In'}
                    </button>
                </form>
            </div>
        </div>
    );
}
