# Mobile Coding Connector - Design Plan

## Overview

A mobile-first coding agent interface that allows users to manage remote development workspaces, interact with AI agents, run terminal commands, and manage port forwarding - all from a mobile device.

## Core Features

### 1. Workspace Management
- **List workspaces**: View all available workspaces with status indicators (running/stopped/error)
- **Create workspace**: Quick workspace creation with template selection
- **Switch workspace**: Easy switching between multiple workspaces
- **Workspace status**: Real-time status updates (CPU, memory, disk usage)
- **Delete/Archive workspace**: Cleanup unused workspaces

### 2. AI Agent Interaction
- **Prompt input**: Large, mobile-friendly text input for sending prompts
- **Conversation history**: Scrollable chat-like interface showing agent responses
- **Agent status**: Visual indicator showing if agent is thinking/executing/idle
- **Stop/Cancel**: Ability to interrupt long-running agent tasks
- **Context awareness**: Show current file/directory context to agent

### 3. Terminal Access
- **Full terminal**: Interactive terminal with keyboard support
- **Command history**: Quick access to recent commands
- **Multiple sessions**: Support for multiple terminal tabs
- **Output streaming**: Real-time output display
- **Mobile keyboard**: Optimized keyboard with common shortcuts (Ctrl+C, Tab, etc.)

### 4. Port Forwarding Management
- **List forwarded ports**: View all active port forwards
- **Add port forward**: Quick setup for new port forwards
- **Public URLs**: Generate shareable URLs for forwarded ports
- **Status monitoring**: Connection status and traffic indicators
- **Quick preview**: In-app browser for previewing forwarded services

## UI/UX Design

### Navigation Structure

```
┌─────────────────────────────────────────┐
│  [≡] Workspace Name          [⚙] [👤]  │  <- Top Bar
├─────────────────────────────────────────┤
│                                         │
│                                         │
│           Main Content Area             │
│                                         │
│                                         │
│                                         │
├─────────────────────────────────────────┤
│  [🏠]    [🤖]    [>_]    [🔗]    [📁]  │  <- Bottom Nav
│  Home   Agent  Terminal  Ports   Files  │
└─────────────────────────────────────────┘
```

### Screen Layouts

#### Home Screen (Workspace List)
```
┌─────────────────────────────────────────┐
│  Mobile Coding Connector                │
│  ─────────────────────────────────────  │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🟢 my-react-app                 │   │
│  │    React • 2h ago • 512MB       │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🟡 backend-api                  │   │
│  │    Go • 1d ago • 256MB          │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🔴 ml-training                  │   │
│  │    Python • Stopped • --        │   │
│  └─────────────────────────────────┘   │
│                                         │
│            [+ New Workspace]            │
│                                         │
└─────────────────────────────────────────┘
```

#### Agent Chat Screen
```
┌─────────────────────────────────────────┐
│  [←] Agent Chat          [⋮] Context   │
├─────────────────────────────────────────┤
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 👤 Add a login page with OAuth  │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🤖 I'll create a login page     │   │
│  │    with Google OAuth...         │   │
│  │                                 │   │
│  │    ✓ Created LoginPage.tsx      │   │
│  │    ✓ Added OAuth config         │   │
│  │    ○ Installing dependencies... │   │
│  └─────────────────────────────────┘   │
│                                         │
├─────────────────────────────────────────┤
│  ┌─────────────────────────────────┐   │
│  │ Type your prompt...         [→] │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

#### Terminal Screen
```
┌─────────────────────────────────────────┐
│  [←] Terminal    [Tab1] [Tab2] [+]     │
├─────────────────────────────────────────┤
│  ~/my-react-app $                       │
│  npm run dev                            │
│                                         │
│  > my-react-app@0.1.0 dev               │
│  > vite                                 │
│                                         │
│    VITE v5.0.0  ready in 234 ms         │
│                                         │
│    ➜  Local:   http://localhost:5173/   │
│    ➜  Network: http://192.168.1.5:5173/ │
│                                         │
│  ~/my-react-app $ _                     │
│                                         │
├─────────────────────────────────────────┤
│ [Tab] [Ctrl] [↑] [↓] [C] [D] [L] [⌨️]  │
└─────────────────────────────────────────┘
```

#### Port Forwarding Screen
```
┌─────────────────────────────────────────┐
│  [←] Port Forwarding                   │
├─────────────────────────────────────────┤
│                                         │
│  Active Forwards                        │
│  ─────────────────────────────────────  │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🟢 :5173 → Frontend Dev         │   │
│  │    https://abc123.tunnel.dev    │   │
│  │    [Copy] [Open] [Stop]         │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🟢 :3000 → API Server           │   │
│  │    https://xyz789.tunnel.dev    │   │
│  │    [Copy] [Open] [Stop]         │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ + Add Port Forward              │   │
│  │   Port: [____] Label: [______]  │   │
│  │              [Forward]          │   │
│  └─────────────────────────────────┘   │
│                                         │
└─────────────────────────────────────────┘
```

## Technical Architecture

### Frontend Components

```
MobileCodingConnector/
├── MobileCodingConnector.tsx    # Main container with routing
├── MobileCodingConnector.css    # Styles
├── components/
│   ├── WorkspaceList.tsx        # Home screen with workspace list
│   ├── WorkspaceCard.tsx        # Individual workspace card
│   ├── AgentChat.tsx            # AI agent interaction
│   ├── ChatMessage.tsx          # Individual chat message
│   ├── TerminalView.tsx         # Terminal interface
│   ├── PortForwarding.tsx       # Port forward management
│   └── BottomNav.tsx            # Bottom navigation
└── hooks/
    └── useWorkspace.ts          # Workspace state management
```

### API Endpoints (Server-side)

```
GET    /api/workspaces              # List all workspaces
POST   /api/workspaces              # Create workspace
DELETE /api/workspaces/:id          # Delete workspace
GET    /api/workspaces/:id/status   # Get workspace status

POST   /api/workspaces/:id/agent    # Send prompt to agent
GET    /api/workspaces/:id/agent/stream  # SSE for agent responses
POST   /api/workspaces/:id/agent/stop    # Stop agent execution

WS     /api/workspaces/:id/terminal # WebSocket for terminal
POST   /api/workspaces/:id/terminal/resize  # Resize terminal

GET    /api/workspaces/:id/ports    # List port forwards
POST   /api/workspaces/:id/ports    # Create port forward
DELETE /api/workspaces/:id/ports/:port  # Stop port forward
```

### State Management

```typescript
interface AppState {
    // Current workspace
    currentWorkspace: Workspace | null;
    workspaces: Workspace[];
    
    // Agent state
    agentStatus: 'idle' | 'thinking' | 'executing';
    chatHistory: ChatMessage[];
    
    // Terminal state
    terminalSessions: TerminalSession[];
    activeTerminal: string | null;
    
    // Port forwarding
    portForwards: PortForward[];
}

interface Workspace {
    id: string;
    name: string;
    type: string;  // react, go, python, etc.
    status: 'running' | 'stopped' | 'error';
    lastAccessed: Date;
    resources: {
        cpu: number;
        memory: number;
        disk: number;
    };
}

interface ChatMessage {
    id: string;
    role: 'user' | 'agent';
    content: string;
    timestamp: Date;
    actions?: AgentAction[];
}

interface AgentAction {
    type: 'file_create' | 'file_edit' | 'command' | 'install';
    status: 'pending' | 'running' | 'done' | 'error';
    description: string;
}

interface PortForward {
    localPort: number;
    label: string;
    publicUrl: string;
    status: 'active' | 'connecting' | 'error';
    traffic: {
        bytesIn: number;
        bytesOut: number;
    };
}
```

## Design Principles

1. **Mobile-First**: All interactions designed for touch, with large tap targets (min 44px)
2. **Offline Awareness**: Clear indicators when connection is lost, queue actions when possible
3. **Progressive Disclosure**: Show essential info first, details on demand
4. **Gesture Support**: Swipe to switch tabs, pull to refresh, long-press for context menus
5. **Dark Mode Default**: Developer-friendly dark theme with optional light mode

## Color Palette

```css
/* Primary Colors */
--primary: #60a5fa;        /* Blue - primary actions */
--primary-dark: #3b82f6;   /* Darker blue - hover states */

/* Status Colors */
--success: #22c55e;        /* Green - running, success */
--warning: #f59e0b;        /* Amber - starting, warning */
--error: #ef4444;          /* Red - stopped, error */

/* Background Colors */
--bg-primary: #0f172a;     /* Deep navy - main background */
--bg-secondary: #1e293b;   /* Slate - cards, panels */
--bg-tertiary: #334155;    /* Lighter slate - inputs */

/* Text Colors */
--text-primary: #f1f5f9;   /* Near white - primary text */
--text-secondary: #94a3b8; /* Gray - secondary text */
--text-muted: #64748b;     /* Darker gray - muted text */

/* Border Colors */
--border: #334155;         /* Subtle borders */
--border-focus: #60a5fa;   /* Focus state borders */
```

## Implementation Phases

### Phase 1: Core UI Shell
- Bottom navigation
- Workspace list view
- Basic workspace card

### Phase 2: Agent Integration
- Chat interface
- Message components
- Agent status indicators

### Phase 3: Terminal
- Terminal view
- Mobile keyboard shortcuts
- Multiple sessions

### Phase 4: Port Forwarding
- Port list view
- Add/remove forwards
- Public URL generation

### Phase 5: Polish
- Animations and transitions
- Error handling
- Offline support
- Performance optimization
