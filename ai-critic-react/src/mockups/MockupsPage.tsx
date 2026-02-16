import { Routes, Route, useParams, Link } from 'react-router-dom';
import { MobileWorkspace } from './MobileWorkspace';
import { MobileCodingConnector } from './v2';
import { ServerFiles } from './ServerFiles';
import { NewTerminalMockup } from './NewTerminalMockup';
import { NoZoomingInputDemo } from './NoZoomingInputDemo';
import { ActionItemCardMockup } from './ActionItemCardMockup';
import './MockupsPage.css';

interface MockupItem {
    id: string;
    name: string;
    description: string;
    component: React.ComponentType;
}

const mockups: MockupItem[] = [
    {
        id: 'action-card',
        name: 'Action Item Card',
        description: 'Mobile-optimized action card with expandable script and streaming logs',
        component: ActionItemCardMockup,
    },
    {
        id: 'new-terminal',
        name: 'New Terminal',
        description: 'Mobile-optimized terminal with keyboard compatibility',
        component: NewTerminalMockup,
    },
    {
        id: 'mobile-workspace',
        name: 'Mobile Workspace',
        description: 'A VS Code-like mobile-first workspace manager with file tree, tabs, and code editor',
        component: MobileWorkspace,
    },
    {
        id: 'v2',
        name: 'Mobile Coding Connector (v2)',
        description: 'Full mobile coding agent with workspace management, AI chat, terminal, and port forwarding',
        component: MobileCodingConnector,
    },
    {
        id: 'server-files',
        name: 'Server Files',
        description: 'Redesigned mobile-first file browser with tree view, file actions, and navigation',
        component: ServerFiles,
    },
    {
        id: 'nozoom-input',
        name: 'No Zooming Input',
        description: 'Demo of inputs that prevent iOS Safari zooming',
        component: NoZoomingInputDemo,
    },
];

function MockupsList() {
    return (
        <div className="mockups-page">
            <div className="mockups-header">
                <h1>Mockups</h1>
            </div>

            <div className="mockups-list">
                {mockups.map(mockup => (
                    <Link key={mockup.id} to={`/mockups/${mockup.id}`} className="mockup-card">
                        <span className="mockup-card-icon">📱</span>
                        <span className="mockup-card-title">{mockup.name}</span>
                        <span className="mockup-card-arrow">→</span>
                    </Link>
                ))}
            </div>
        </div>
    );
}

function MockupView() {
    const { mockupId } = useParams<{ mockupId: string }>();
    const activeMockup = mockups.find(m => m.id === mockupId);

    if (!activeMockup) {
        return (
            <div className="mockups-page">
                <div className="mockups-header">
                    <h1>Mockup Not Found</h1>
                    <Link to="/mockups" className="mockup-back-btn">← Back to Mockups</Link>
                </div>
            </div>
        );
    }

    // For v2, render without the header wrapper since it has its own navigation
    if (mockupId === 'v2') {
        return (
            <Routes>
                <Route index element={<MobileCodingConnector />} />
                <Route path=":workspaceId" element={<MobileCodingConnector />} />
            </Routes>
        );
    }

    const MockupComponent = activeMockup.component;
    return (
        <div className="mockup-view">
            <div className="mockup-view-header">
                <Link to="/mockups" className="mockup-back-btn">
                    ← Back to Mockups
                </Link>
                <span className="mockup-view-title">{activeMockup.name}</span>
            </div>
            <div className="mockup-view-content">
                <MockupComponent />
            </div>
        </div>
    );
}

export function MockupsPage() {
    return (
        <Routes>
            <Route index element={<MockupsList />} />
            <Route path=":mockupId/*" element={<MockupView />} />
        </Routes>
    );
}
