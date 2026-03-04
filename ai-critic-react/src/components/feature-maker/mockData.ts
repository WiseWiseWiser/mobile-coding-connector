import type { FlowStatus, Insight } from './types';

export const flowSteps: FlowStatus[] = [
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

export const mockFeatureRequest = {
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

export const mockArchitecturalDecisions = [
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

export const mockSubAgents = [
    { name: 'Architect Agent', role: 'Design system architecture', status: 'idle' },
    { name: 'Coder Agent', role: 'Implement feature code', status: 'idle' },
    { name: 'Verifier Agent', role: 'Test and validate implementation', status: 'idle' },
];

export const mockMainAgent = {
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

export const mockInsights: Insight[] = [
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
