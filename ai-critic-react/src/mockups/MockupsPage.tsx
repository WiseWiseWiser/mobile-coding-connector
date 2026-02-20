import { Routes, Route, useParams, Link } from 'react-router-dom';
import { MobileWorkspace } from './MobileWorkspace';
import { MobileCodingConnector } from './v2';
import { ServerFiles } from './ServerFiles';
import { NewTerminalMockup } from './NewTerminalMockup';
import { V2TerminalMockup } from './V2TerminalMockup';
import { ExitedTerminalTest } from './ExitedTerminalTest';
import { NoZoomingInputDemo } from './NoZoomingInputDemo';
import { PureTerminalMockup } from './PureTerminalMockup';
import { ScrollbarMockup } from './ScrollbarMockup';
import { ActionItemCardMockup } from './ActionItemCardMockup';
import { FeatureMakerMockup } from './FeatureMakerMockup';
import { PathInputDemo } from './PathInputDemo';
import { CommandInputMockup } from './CommandInputMockup';
import { XtermQuickTerminal } from './CommandSuccessTerminal';
import { CustomizeQuickTerminal } from './CustomizeQuickTerminal';
import { SessionsSectionMockup } from './SessionsSectionMockup';
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
        id: 'pure-terminal',
        name: 'Pure Terminal View',
        description: 'Core terminal component without tabs or quick input - the reusable building block',
        component: PureTerminalMockup,
    },
    {
        id: 'custom-scrollbar',
        name: 'Custom Scrollbar',
        description: 'iOS-optimized scrollbar with drag and swipe support for horizontal/vertical scrolling',
        component: ScrollbarMockup,
    },
    {
        id: 'new-terminal',
        name: 'New Terminal',
        description: 'Mobile-optimized terminal with keyboard compatibility',
        component: NewTerminalMockup,
    },
    {
        id: 'v2-terminal',
        name: 'V2 Terminal',
        description: '12 terminal states demonstrating different UX patterns for mobile terminals',
        component: V2TerminalMockup,
    },
    {
        id: 'exited-terminal-test',
        name: 'Exited Terminal Test',
        description: 'Test page with only an exited terminal to debug refresh issue',
        component: ExitedTerminalTest,
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
    {
        id: 'feature-maker',
        name: 'FeatureMaker',
        description: 'AI-driven feature implementation flow with driver agent and subagents',
        component: FeatureMakerMockup,
    },
    {
        id: 'path-input',
        name: 'PathInput',
        description: 'Editable path input with no-zoom for iOS',
        component: PathInputDemo,
    },
    {
        id: 'command-input',
        name: 'Command Input',
        description: 'Command input with history dropdown, fuzzy search, and keyboard navigation',
        component: CommandInputMockup,
    },
    {
        id: 'command-success-terminal',
        name: 'Xterm Quick Terminal',
        description: 'Interactive terminal with xterm, input bar, shortcuts, and fake bash server',
        component: XtermQuickTerminal,
    },
    {
        id: 'customize-quick-terminal',
        name: 'Customize Quick Terminal',
        description: 'Custom native terminal (iOS optimized) vs xterm-based terminal',
        component: CustomizeQuickTerminal,
    },
    {
        id: 'sessions-section',
        name: 'Sessions Section',
        description: 'Pure-view component for displaying and managing agent sessions',
        component: SessionsSectionMockup,
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
