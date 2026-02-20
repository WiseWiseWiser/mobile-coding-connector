import { useState, useRef, useEffect } from 'react';
import { MockupPageContainer } from './MockupPageContainer';
import { PathInput } from '../pure-view/PathInput';
import { getFakeLLMServer, type FakeLLMSession, type StreamEvent } from './fake';
import './FeatureMakerMockup.css';

type FlowStep = 'understanding' | 'clarifying' | 'implementing' | 'verifying' | 'completed';

interface FlowStatus {
    step: FlowStep;
    title: string;
    description: string;
    status: 'pending' | 'active' | 'completed' | 'error';
}

const flowSteps: FlowStatus[] = [
    {
        step: 'understanding',
        title: 'Understanding the Request',
        description: 'Parse and comprehend the feature request, identify goals and constraints',
        status: 'completed',
    },
    {
        step: 'clarifying',
        title: 'Clarify Architectural Decisions',
        description: 'Determine API contracts, data models, and integration points',
        status: 'active',
    },
    {
        step: 'implementing',
        title: 'Implementing with Coding',
        description: 'Write code, create components, and integrate with existing systems',
        status: 'pending',
    },
    {
        step: 'verifying',
        title: 'Verify the Implementation',
        description: 'Run tests, validate against requirements, ensure quality',
        status: 'pending',
    },
];

const mockFeatureRequest = {
    title: 'Add Feature Request Management to Mobile App',
    description: 'Implement a feature request system where users can submit, upvote, and track feature requests. The system should include a RESTful API, real-time updates via WebSocket, and a mobile-optimized UI with push notifications.',
    details: [
        'User authentication and authorization',
        'Feature request CRUD operations',
        'Upvoting system with leaderboard',
        'Real-time updates using WebSocket',
        'Push notification integration',
        'Admin dashboard for managing requests',
    ],
    priority: 'high',
    estimatedComplexity: 'medium',
};

const mockArchitecturalDecisions = [
    {
        area: 'API Design',
        decision: 'RESTful API with JSON response format',
        rationale: 'Familiar pattern, easy to test, good tooling support',
    },
    {
        area: 'Database',
        decision: 'PostgreSQL with Prisma ORM',
        rationale: 'Relational data fits well, strong typing with Prisma',
    },
    {
        area: 'Real-time',
        decision: 'Socket.io for WebSocket communication',
        rationale: 'Proven reliability, rooms support for different events',
    },
    {
        area: 'Authentication',
        decision: 'JWT tokens with refresh token rotation',
        rationale: 'Stateless, scales well, industry standard',
    },
];

const mockSubAgents = [
    { name: 'Architect Agent', role: 'Design system architecture', status: 'idle' },
    { name: 'Coder Agent', role: 'Implement feature code', status: 'idle' },
    { name: 'Verifier Agent', role: 'Test and validate implementation', status: 'idle' },
];

const mockMainAgent = {
    name: 'Driver Agent',
    systemPrompt: `# Role
You are the Driver Agent responsible for orchestrating feature implementation using subagents.

# Workflow
1. **Understand**: Parse the feature request and identify key requirements
2. **Clarify**: Determine architectural decisions with the Architect Agent
3. **Implement**: Delegate coding tasks to the Coder Agent
4. **Verify**: Have the Verifier Agent validate the implementation

# Examples
- When receiving a feature request, first break it down into manageable tasks
- Delegate appropriate subtasks to specialized subagents
- Aggregate results and present coherent feedback to the user
- Always verify before marking a step as complete`,

    tools: [
        { name: 'delegate_task', description: 'Delegate a subtask to a specific subagent' },
        { name: 'ask_user', description: 'Ask the user for clarification or confirmation' },
        { name: 'invoke_tool', description: 'Execute a tool on the host system' },
        { name: 'report_status', description: 'Report current progress to the user' },
    ],
};

type InsightType = 'decision' | 'action' | 'question';

interface Insight {
    id: string;
    type: InsightType;
    title: string;
    description: string;
    agent: string;
}

const mockInsights: Insight[] = [
    {
        id: '1',
        type: 'decision',
        title: 'Chose PostgreSQL over MongoDB',
        description: 'Relational model better fits the structured feature request data with clear relationships between users, requests, and votes.',
        agent: 'Architect Agent',
    },
    {
        id: '2',
        type: 'question',
        title: 'How to handle duplicate feature requests?',
        description: 'Should duplicate requests be merged automatically or require manual review? Current approach: merge automatically if similarity > 90%.',
        agent: 'Architect Agent',
    },
    {
        id: '3',
        type: 'action',
        title: 'Add WebSocket connection pooling',
        description: 'Production deployment requires connection pooling for WebSocket to handle 1000+ concurrent connections efficiently.',
        agent: 'Coder Agent',
    },
];

type DriverAgentStatus = 'idle' | 'running' | 'paused' | 'finished';

export function FeatureMakerMockup() {
    const [currentStep] = useState<FlowStep>('clarifying');
    const [showDoc, setShowDoc] = useState(false);
    const [selectedInsight, setSelectedInsight] = useState<Insight | null>(null);
    const [projectPath, setProjectPath] = useState('/workspace/feature-request-app');
    const [driverAgentStatus, setDriverAgentStatus] = useState<DriverAgentStatus>('idle');
    const [streamEvents, setStreamEvents] = useState<StreamEvent[]>([]);
    const [progress, setProgress] = useState(0);
    const sessionRef = useRef<FakeLLMSession | null>(null);
    const streamEventsRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        if (streamEventsRef.current) {
            streamEventsRef.current.scrollTop = streamEventsRef.current.scrollHeight;
        }
    }, [streamEvents]);

    const handleStart = () => {
        setDriverAgentStatus('running');
        setStreamEvents([]);
        setProgress(0);

        const server = getFakeLLMServer();
        const session = server.startStream(mockFeatureRequest.description, projectPath);
        sessionRef.current = session;

        session.onStart(() => {
            setStreamEvents(prev => [...prev, { type: 'start', message: 'Starting driver agent...', timestamp: Date.now() }]);
        });

        session.onStep((event) => {
            setStreamEvents(prev => [...prev, event]);
            if (event.progress) setProgress(event.progress);
        });

        session.onDone((event) => {
            setStreamEvents(prev => [...prev, event]);
            setDriverAgentStatus('finished');
            setProgress(100);
            sessionRef.current = null;
        });

        session.onAborted((event) => {
            setStreamEvents(prev => [...prev, event]);
            setDriverAgentStatus('idle');
            sessionRef.current = null;
        });
    };

    const handlePause = () => setDriverAgentStatus('paused');
    
    const handleResume = () => {
        setDriverAgentStatus('running');
    };

    const handleAbort = () => {
        if (sessionRef.current) {
            sessionRef.current.abort();
            sessionRef.current = null;
        }
        setDriverAgentStatus('idle');
        setProgress(0);
    };

    return (
        <MockupPageContainer
            title="FeatureMaker"
            description="AI-driven feature implementation flow with subagents"
        >
            <div className="feature-maker">
                <div className="fm-header">
                    <button 
                        className={`fm-toggle-doc ${showDoc ? 'active' : ''}`}
                        onClick={() => setShowDoc(!showDoc)}
                    >
                        {showDoc ? '▼' : '▶'} Flow Documentation
                    </button>
                </div>

                {showDoc && (
                    <div className="fm-doc-panel">
                        <h3>FeatureMaker Flow</h3>
                        <p>
                            FeatureMaker is an AI-driven workflow for implementing features using a driver agent 
                            that orchestrates dedicated subagents. The flow follows a 4-step process:
                        </p>
                        <ol>
                            <li>
                                <strong>Understanding the Request</strong> - Parse and comprehend the feature request, 
                                identify goals, constraints, and success criteria
                            </li>
                            <li>
                                <strong>Clarify Architectural Decisions</strong> - Determine API contracts, data models, 
                                integration points, and technical approach
                            </li>
                            <li>
                                <strong>Implementing with Coding</strong> - Write code, create components, 
                                and integrate with existing systems using subagents
                            </li>
                            <li>
                                <strong>Verify the Implementation</strong> - Run tests, validate against requirements, 
                                ensure code quality and performance
                            </li>
                        </ol>
                        <p className="fm-doc-note">
                            A <strong>Driver Agent</strong> orchestrates this flow, delegating subprocesses to 
                            dedicated subagents (Architect, Coder, Verifier).
                        </p>
                    </div>
                )}

                <div className="fm-content">
                    <div className="fm-feature-request">
                        <div className="fm-fr-header">
                            <h3>Feature Request</h3>
                            <div className="fm-fr-meta">
                                <span className={`fm-priority ${mockFeatureRequest.priority}`}>
                                    {mockFeatureRequest.priority}
                                </span>
                                <span className="fm-complexity">
                                    Complexity: {mockFeatureRequest.estimatedComplexity}
                                </span>
                            </div>
                        </div>
                        <div className="fm-fr-content">
                            <h4>{mockFeatureRequest.title}</h4>
                            <p>{mockFeatureRequest.description}</p>
                            <div className="fm-fr-details">
                                <h5>Key Requirements:</h5>
                                <ul>
                                    {mockFeatureRequest.details.map((detail, i) => (
                                        <li key={i}>{detail}</li>
                                    ))}
                                </ul>
                            </div>
                        </div>
                    </div>

                    <div className="fm-flow-stages">
                        {flowSteps.map((step, index) => (
                            <div 
                                key={step.step}
                                className={`fm-flow-stage ${step.status} ${step.step === currentStep ? 'current' : ''}`}
                            >
                                <div className="fm-stage-indicator">
                                    {step.status === 'completed' ? '✓' : step.status === 'error' ? '✗' : index + 1}
                                </div>
                                <span className="fm-stage-title">{step.title}</span>
                            </div>
                        ))}
                    </div>

                    <div className="fm-sidebar">
                        <div className="fm-project-path">
                            <PathInput
                                value={projectPath}
                                onChange={setProjectPath}
                                label="Project Directory"
                            />
                        </div>

                        <div className="fm-main-agent">
                            <h3>Main Agent</h3>
                            <div className="fm-agent-name">{mockMainAgent.name}</div>
                            
                            <div className="fm-agent-prompt">
                                <h4>System Prompt</h4>
                                <div className="fm-prompt-content">
                                    {mockMainAgent.systemPrompt.split('\n\n').map((section, i) => {
                                        const lines = section.split('\n');
                                        const title = lines[0].replace(/^#\s*/, '');
                                        const content = lines.slice(1).join('\n');
                                        return (
                                            <div key={i} className="fm-prompt-section">
                                                <span className="fm-prompt-section-title">{title}</span>
                                                <span className="fm-prompt-section-content">{content}</span>
                                            </div>
                                        );
                                    })}
                                </div>
                            </div>

                            <div className="fm-agent-tools">
                                <h4>Available Tools</h4>
                                <div className="fm-tools-list">
                                    {mockMainAgent.tools.map((tool, i) => (
                                        <div key={i} className="fm-tool-item">
                                            <span className="fm-tool-name">{tool.name}</span>
                                            <span className="fm-tool-desc">{tool.description}</span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </div>

                        <div className="fm-subagents">
                            <h3>Subagents</h3>
                            <div className="fm-agent-list">
                                {mockSubAgents.map(agent => (
                                    <div key={agent.name} className={`fm-agent ${agent.status}`}>
                                        <span className="fm-agent-status-dot"></span>
                                        <span className="fm-agent-name">{agent.name}</span>
                                        <span className="fm-agent-role">{agent.role}</span>
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>

                    <div className="fm-panels">
                        <div className="fm-run-section">
                            <div className="fm-run-controls">
                                <div className="fm-driver-control">
                                    {driverAgentStatus === 'idle' && (
                                        <button className="fm-driver-btn fm-driver-start" onClick={handleStart}>
                                            ▶ Start the driver agent to implement the feature
                                        </button>
                                    )}
                                    {driverAgentStatus === 'running' && (
                                        <button className="fm-driver-btn fm-driver-pause" onClick={handlePause}>
                                            ⏸ Pause
                                        </button>
                                    )}
                                    {driverAgentStatus === 'paused' && (
                                        <div className="fm-driver-paused-controls">
                                            <button className="fm-driver-btn fm-driver-resume" onClick={handleResume}>
                                                ▶ Resume
                                            </button>
                                            <button className="fm-driver-btn fm-driver-abort" onClick={handleAbort}>
                                                ⏹ Abort
                                            </button>
                                        </div>
                                    )}
                                    {driverAgentStatus === 'finished' && (
                                        <div className="fm-driver-finished">
                                            <span className="fm-finished-badge">✓ Finished</span>
                                        </div>
                                    )}
                                </div>
                            </div>
                            {(driverAgentStatus === 'running' || driverAgentStatus === 'paused' || driverAgentStatus === 'finished') && (
                                <div className="fm-run-progress">
                                    <div className="fm-progress-bar">
                                        <div className="fm-progress-fill" style={{ width: `${progress}%` }}></div>
                                    </div>
                                    <div className="fm-stream-events" ref={streamEventsRef}>
                                        {streamEvents.map((event, i) => (
                                            <div key={i} className={`fm-stream-event fm-event-type-${event.type}`}>
                                                {event.step && <span className="fm-event-step">[{event.step}]</span>}
                                                <span className="fm-event-message">{event.message}</span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>

                        <div className="fm-flow-section completed">
                            <div className="fm-section-header">
                                <span className="fm-section-num">1</span>
                                <span className="fm-section-title">Understanding the Request</span>
                                <span className="fm-section-status">✓</span>
                            </div>
                            <div className="fm-section-content">
                                <p>Parsed feature request: "Add Feature Request Management to Mobile App"</p>
                                <p>Identified 6 key requirements including authentication, CRUD, upvoting, WebSocket, push notifications, and admin dashboard.</p>
                            </div>
                        </div>

                        <div className="fm-flow-section active">
                            <div className="fm-section-header">
                                <span className="fm-section-num">2</span>
                                <span className="fm-section-title">Clarify Architectural Decisions</span>
                                <span className="fm-section-status">●</span>
                            </div>
                            <div className="fm-section-content">
                                <div className="fm-decisions">
                                    {mockArchitecturalDecisions.map((ad, i) => (
                                        <div key={i} className="fm-decision">
                                            <span className="fm-decision-area">{ad.area}</span>
                                            <span className="fm-decision-value">{ad.decision}</span>
                                            <span className="fm-decision-rationale">{ad.rationale}</span>
                                        </div>
                                    ))}
                                </div>

                                <div className="fm-insights-section">
                                    <h4>Agent Insights</h4>
                                    <div className="fm-insights-list">
                                        {mockInsights.map(insight => (
                                            <div 
                                                key={insight.id} 
                                                className={`fm-insight-item ${insight.type}`}
                                                onClick={() => setSelectedInsight(insight)}
                                            >
                                                <span className="fm-insight-icon">
                                                    {insight.type === 'decision' ? '✓' : insight.type === 'action' ? '⚡' : '?'}
                                                </span>
                                                <div className="fm-insight-info">
                                                    <span className="fm-insight-title">{insight.title}</span>
                                                    <span className="fm-insight-agent">{insight.agent}</span>
                                                </div>
                                                <span className="fm-insight-arrow">›</span>
                                            </div>
                                        ))}
                                    </div>
                                </div>

                                <div className="fm-clarifying-status">
                                    <div className="fm-clarifying-indicator">
                                        <span className="fm-spinner"></span>
                                        <span>Architect Agent is analyzing requirements...</span>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div className="fm-flow-section pending">
                            <div className="fm-section-header">
                                <span className="fm-section-num">3</span>
                                <span className="fm-section-title">Implementing with Coding</span>
                                <span className="fm-section-status">○</span>
                            </div>
                            <div className="fm-section-content">
                                <div className="fm-implementation-tasks">
                                    <div className="fm-task completed">
                                        <span className="fm-task-check">✓</span>
                                        <span>Database schema design</span>
                                    </div>
                                    <div className="fm-task completed">
                                        <span className="fm-task-check">✓</span>
                                        <span>API endpoint definitions</span>
                                    </div>
                                    <div className="fm-task pending">
                                        <span className="fm-task-check">○</span>
                                        <span>Implementing request handlers</span>
                                    </div>
                                    <div className="fm-task pending">
                                        <span className="fm-task-check">○</span>
                                        <span>WebSocket integration</span>
                                    </div>
                                    <div className="fm-task pending">
                                        <span className="fm-task-check">○</span>
                                        <span>Frontend components</span>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div className="fm-flow-section pending">
                            <div className="fm-section-header">
                                <span className="fm-section-num">4</span>
                                <span className="fm-section-title">Verify the Implementation</span>
                                <span className="fm-section-status">○</span>
                            </div>
                            <div className="fm-section-content">
                                <div className="fm-verification-checks">
                                    <div className="fm-check pending">
                                        <span className="fm-check-icon">○</span>
                                        <span>Run unit tests</span>
                                    </div>
                                    <div className="fm-check pending">
                                        <span className="fm-check-icon">○</span>
                                        <span>Run integration tests</span>
                                    </div>
                                    <div className="fm-check pending">
                                        <span className="fm-check-icon">○</span>
                                        <span>Verify code coverage</span>
                                    </div>
                                    <div className="fm-check pending">
                                        <span className="fm-check-icon">○</span>
                                        <span>Run linter</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    {selectedInsight && (
                        <div className="fm-insight-modal-overlay" onClick={() => setSelectedInsight(null)}>
                            <div className="fm-insight-modal" onClick={e => e.stopPropagation()}>
                                <div className="fm-modal-header">
                                    <span className={`fm-modal-type ${selectedInsight.type}`}>
                                        {selectedInsight.type === 'decision' ? 'Decision' : selectedInsight.type === 'action' ? 'Action' : 'Question'}
                                    </span>
                                    <button className="fm-modal-close" onClick={() => setSelectedInsight(null)}>×</button>
                                </div>
                                <div className="fm-modal-body">
                                    <h3>{selectedInsight.title}</h3>
                                    <p>{selectedInsight.description}</p>
                                    <div className="fm-modal-agent">
                                        <span className="fm-modal-agent-label">From:</span>
                                        <span className="fm-modal-agent-name">{selectedInsight.agent}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </MockupPageContainer>
    );
}
