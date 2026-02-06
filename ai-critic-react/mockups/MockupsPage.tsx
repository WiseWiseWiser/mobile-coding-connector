import { Routes, Route, useNavigate, useParams, Link } from 'react-router-dom';
import { MobileWorkspace } from './MobileWorkspace';
import { MobileCodingConnector } from './v2';
import './MockupsPage.css';

interface MockupItem {
    id: string;
    name: string;
    description: string;
    component: React.ComponentType;
}

const mockups: MockupItem[] = [
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
];

function MockupsList() {
    const navigate = useNavigate();

    return (
        <div className="mockups-page">
            <div className="mockups-header">
                <h1>Design Mockups</h1>
                <p>Prototypes and design explorations for mobile-first features</p>
                <div className="mockups-dev-badge">DEV MODE ONLY</div>
            </div>

            <div className="mockups-grid">
                {mockups.map(mockup => (
                    <div 
                        key={mockup.id}
                        className="mockup-card"
                        onClick={() => navigate(`/mockups/${mockup.id}`)}
                    >
                        <div className="mockup-card-preview">
                            <div className="mockup-card-icon">📱</div>
                        </div>
                        <div className="mockup-card-info">
                            <h3>{mockup.name}</h3>
                            <p>{mockup.description}</p>
                        </div>
                    </div>
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
