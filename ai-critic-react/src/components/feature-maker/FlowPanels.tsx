import type { Insight } from './types';
import { mockArchitecturalDecisions, mockInsights } from './mockData';

interface FlowPanelsProps {
    selectedInsight: Insight | null;
    onSelectInsight: (insight: Insight) => void;
}

export function FlowPanels({ selectedInsight: _, onSelectInsight }: FlowPanelsProps) {
    return (
        <>
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
                                    onClick={() => onSelectInsight(insight)}
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
        </>
    );
}
