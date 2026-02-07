import { BrowserRouter as Router, Routes, Route, Link, Outlet, Navigate } from 'react-router-dom';
import { lazy, Suspense, useState, useEffect } from 'react';
import AppGen from './AppGen';
import CodeReview from './CodeReview';
import { AppLayout } from './components/layout';
import { MobileCodingConnector, LoginPage, SetupPage, V2Provider, WorkspaceListView, DiagnoseView, SettingsView, ExportPage, ImportPage, CloudflareSettingsView, GitSettings, CloneRepoView, UploadFileView, TerminalView, AgentLayout, AgentPickerRoute, SessionListRoute, AgentChatRoute, PortsLayout, PortListRoute, CloudflareDiagnosticsRoute, PortDiagnoseRoute, FilesLayout, FilesTabLayout, CheckpointListRoute, CreateCheckpointRoute, CheckpointDetailRoute, FileBrowserRoute, FileContentRoute } from './v2';
import { checkAuth, AuthCheckStatuses } from './api/auth';
import './App.css';

// Conditionally import mockups only in dev mode
const MockupsPage = import.meta.env.DEV
    ? lazy(() => import('../mockups/MockupsPage').then(m => ({ default: m.MockupsPage })))
    : null;

function Home() {
    return (
        <div className="home-container">
            <div className="home-hero">
                <img src="/ai-critic.svg" alt="AI Critic" className="home-logo" />
                <h1 className="home-title">AI Critic</h1>
                <p className="home-subtitle">
                    Intelligent code review powered by AI
                </p>
                <div className="home-actions">
                    <Link to="/v1" className="home-btn home-btn-primary">
                        Start Code Review
                    </Link>
                    <Link to="/v1/gen" className="home-btn home-btn-secondary">
                        Code Generator
                    </Link>
                </div>
            </div>

            <div className="home-features">
                <div className="feature-card">
                    <div className="feature-icon">📝</div>
                    <h3>Smart Reviews</h3>
                    <p>AI-powered code analysis that catches issues before they reach production</p>
                </div>
                <div className="feature-card">
                    <div className="feature-icon">🔍</div>
                    <h3>Diff Viewer</h3>
                    <p>Visual diff comparison with syntax highlighting and inline comments</p>
                </div>
                <div className="feature-card">
                    <div className="feature-icon">💻</div>
                    <h3>Terminal</h3>
                    <p>Built-in terminal for quick command execution without leaving the app</p>
                </div>
                <div className="feature-card">
                    <div className="feature-icon">📱</div>
                    <h3>Mobile Ready</h3>
                    <p>Responsive design that works seamlessly on desktop and mobile devices</p>
                </div>
            </div>
        </div>
    );
}

function About() {
    return (
        <div style={{ textAlign: 'center', padding: '50px' }}>
            <h1>About</h1>
            <p>This is a generic about page.</p>
            <Link to="/v1" style={{ fontSize: '18px', color: '#646cff', textDecoration: 'none' }}>
                Back to Home
            </Link>
        </div>
    );
}

// Main app content with old navigation
function MainApp() {
    return (
        <AppLayout>
            <Routes>
                <Route path="/" element={<CodeReview />} />
                <Route path="/home" element={<Home />} />
                <Route path="/about" element={<About />} />
                <Route path="/gen" element={<AppGen />} />
                {MockupsPage && (
                    <Route 
                        path="/mockups/*" 
                        element={
                            <Suspense fallback={<div style={{ padding: 20, textAlign: 'center' }}>Loading mockups...</div>}>
                                <MockupsPage />
                            </Suspense>
                        } 
                    />
                )}
            </Routes>
        </AppLayout>
    );
}

// Auth states
const AuthStates = {
    Loading: 'loading',
    Authenticated: 'authenticated',
    Unauthenticated: 'unauthenticated',
    NotInitialized: 'not_initialized',
} as const;

type AuthState = typeof AuthStates[keyof typeof AuthStates];

// V2 Layout - handles auth, wraps child routes via Outlet so they share state
function V2Layout() {
    const [authState, setAuthState] = useState<AuthState>(AuthStates.Loading);

    useEffect(() => {
        checkAuth()
            .then(status => {
                if (status === AuthCheckStatuses.Authenticated) {
                    setAuthState(AuthStates.Authenticated);
                } else if (status === AuthCheckStatuses.NotInitialized) {
                    setAuthState(AuthStates.NotInitialized);
                } else {
                    setAuthState(AuthStates.Unauthenticated);
                }
            })
            .catch(() => {
                // Network error - assume authenticated (server might not require auth)
                setAuthState(AuthStates.Authenticated);
            });
    }, []);

    if (authState === AuthStates.Loading) {
        return null;
    }

    if (authState === AuthStates.NotInitialized) {
        return <SetupPage onSetupComplete={() => setAuthState(AuthStates.Unauthenticated)} />;
    }

    if (authState === AuthStates.Unauthenticated) {
        return <LoginPage onLoginSuccess={() => setAuthState(AuthStates.Authenticated)} />;
    }

    // V2Provider holds shared state that survives child route remounts
    return (
        <V2Provider>
            <Outlet />
        </V2Provider>
    );
}

function App() {
    return (
        <Router>
            <Routes>
                {/* Legacy v1 routes */}
                <Route path="/v1/*" element={<MainApp />} />
                {/* Main routes (v2) - layout wraps all child routes */}
                <Route path="/" element={<V2Layout />}>
                    <Route index element={<Navigate to="home" replace />} />
                    {/* Non-project routes: MobileCodingConnector as layout */}
                    <Route element={<MobileCodingConnector />}>
                        <Route path="home">
                            <Route index element={<WorkspaceListView />} />
                            <Route path="diagnose" element={<DiagnoseView />} />
                            <Route path="settings" element={<SettingsView />} />
                            <Route path="settings/export" element={<ExportPage />} />
                            <Route path="settings/import" element={<ImportPage />} />
                            <Route path="settings/cloudflare" element={<CloudflareSettingsView />} />
                            <Route path="settings/git" element={<GitSettings />} />
                            <Route path="clone-repo" element={<CloneRepoView />} />
                            <Route path="upload-file" element={<UploadFileView />} />
                        </Route>
                        <Route path="agent" element={<AgentLayout />}>
                            <Route index element={<AgentPickerRoute />} />
                            <Route path=":agentId" element={<SessionListRoute />} />
                            <Route path=":agentId/:sessionId" element={<AgentChatRoute />} />
                        </Route>
                        <Route path="terminal" element={<TerminalView />} />
                        <Route path="ports" element={<PortsLayout />}>
                            <Route index element={<PortListRoute />} />
                            <Route path="diagnostics" element={<CloudflareDiagnosticsRoute />} />
                            <Route path="port-diagnose/:port" element={<PortDiagnoseRoute />} />
                        </Route>
                        <Route path="files" element={<FilesLayout />}>
                            <Route element={<FilesTabLayout />}>
                                <Route index element={<CheckpointListRoute />} />
                                <Route path="browse" element={<FileBrowserRoute />} />
                                <Route path="browse/*" element={<FileBrowserRoute />} />
                            </Route>
                            <Route path="create-checkpoint" element={<CreateCheckpointRoute />} />
                            <Route path="checkpoint/:checkpointId" element={<CheckpointDetailRoute />} />
                            <Route path="file/*" element={<FileContentRoute />} />
                        </Route>
                    </Route>
                    {/* Project-specific routes: MobileCodingConnector as layout */}
                    <Route path="project/:projectName" element={<MobileCodingConnector />}>
                        <Route index element={<Navigate to="home" replace />} />
                        <Route path="home">
                            <Route index element={<WorkspaceListView />} />
                            <Route path="diagnose" element={<DiagnoseView />} />
                            <Route path="settings" element={<SettingsView />} />
                            <Route path="settings/export" element={<ExportPage />} />
                            <Route path="settings/import" element={<ImportPage />} />
                            <Route path="settings/cloudflare" element={<CloudflareSettingsView />} />
                            <Route path="settings/git" element={<GitSettings />} />
                            <Route path="clone-repo" element={<CloneRepoView />} />
                            <Route path="upload-file" element={<UploadFileView />} />
                        </Route>
                        <Route path="agent" element={<AgentLayout />}>
                            <Route index element={<AgentPickerRoute />} />
                            <Route path=":agentId" element={<SessionListRoute />} />
                            <Route path=":agentId/:sessionId" element={<AgentChatRoute />} />
                        </Route>
                        <Route path="terminal" element={<TerminalView />} />
                        <Route path="ports" element={<PortsLayout />}>
                            <Route index element={<PortListRoute />} />
                            <Route path="diagnostics" element={<CloudflareDiagnosticsRoute />} />
                            <Route path="port-diagnose/:port" element={<PortDiagnoseRoute />} />
                        </Route>
                        <Route path="files" element={<FilesLayout />}>
                            <Route element={<FilesTabLayout />}>
                                <Route index element={<CheckpointListRoute />} />
                                <Route path="browse" element={<FileBrowserRoute />} />
                                <Route path="browse/*" element={<FileBrowserRoute />} />
                            </Route>
                            <Route path="create-checkpoint" element={<CreateCheckpointRoute />} />
                            <Route path="checkpoint/:checkpointId" element={<CheckpointDetailRoute />} />
                            <Route path="file/*" element={<FileContentRoute />} />
                        </Route>
                    </Route>
                </Route>
            </Routes>
        </Router>
    );
}

export default App;
