import { useState } from 'react';
import { MockupPageContainer } from './MockupPageContainer';
import { ConfirmModal, ActionIconSelector, ActionCard } from '../pure-view';
import './ActionItemCardMockup.css';

interface ActionItemCardProps {
    state: 'basic' | 'long-script' | 'running' | 'error' | 'success';
    onDelete?: () => void;
    onEdit?: () => void;
}

const mockActionBasic = {
    id: '1',
    name: 'Build Project',
    icon: '🔨',
    script: 'npm run build',
    running: false,
    exitCode: undefined,
    logs: [],
};

const mockActionLongScript = {
    id: '2',
    name: 'Run Tests',
    icon: '🧪',
    script: `npm test -- --coverage
--watch=false
--verbose
--testPathPattern=src/`,
    running: false,
    exitCode: undefined,
    logs: [],
};

const mockActionRunning = {
    id: '3',
    name: 'Deploy App',
    icon: '🚀',
    script: 'npm run deploy -- --env=production',
    running: true,
    exitCode: undefined,
    logs: [
        { text: '> npm run deploy -- --env=production', error: false },
        { text: '', error: false },
        { text: '> deploy-script@1.0.0 predeploy', error: false },
        { text: '> npm run build', error: false },
        { text: '', error: false },
        { text: '> build-script@1.0.0 build', error: false },
        { text: 'vite v5.0.0 building for production...', error: false },
        { text: '✓ 32 modules transformed.', error: false },
        { text: 'build/assets/index-a1b2c3d4.js  125.43 KB', error: false },
        { text: 'build/index.html                   0.48 KB', error: false },
        { text: 'build/favicon.ico                  0.15 KB', error: false },
        { text: '✨  Built in 3.2s.', error: false },
    ],
};

const mockActionError = {
    id: '4',
    name: 'Lint Code',
    icon: '📋',
    script: 'npm run lint',
    running: false,
    exitCode: 1,
    logs: [
        { text: '> npm run lint', error: false },
        { text: '', error: false },
        { text: '> linter@1.0.0 lint', error: false },
        { text: '', error: false },
        { text: 'src/App.tsx', error: false },
        { text: '  line 10:13  error  Missing semicolon  semi', error: true },
        { text: '  line 15:8   error  Unused variable "foo"  no-unused-vars', error: true },
        { text: '', error: false },
        { text: 'src/utils.ts', error: false },
        { text: '  line 3:1    error  Import order wrong    import/order', error: true },
        { text: '', error: false },
        { text: '✖ 3 problems (3 errors, 0 warnings)', error: true },
    ],
};

const mockActionSuccess = {
    id: '5',
    name: 'Format Code',
    icon: '✨',
    script: 'npm run format',
    running: false,
    exitCode: 0,
    logs: [
        { text: '> npm run format', error: false },
        { text: '', error: false },
        { text: '✓ Formatted 12 files', error: false },
        { text: '  src/App.tsx', error: false },
        { text: '  src/components/Button.tsx', error: false },
        { text: '  src/utils/helpers.ts', error: false },
        { text: '  ...', error: false },
    ],
};

function ActionItemCard({ state, onDelete, onEdit }: ActionItemCardProps & { onDelete?: () => void; onEdit?: () => void }) {
    const getMockData = () => {
        switch (state) {
            case 'basic': return mockActionBasic;
            case 'long-script': return mockActionLongScript;
            case 'running': return mockActionRunning;
            case 'error': return mockActionError;
            case 'success': return mockActionSuccess;
        }
    };

    const action = getMockData();

    return (
        <ActionCard
            name={action.name}
            icon={action.icon}
            script={action.script}
            running={action.running}
            exitCode={action.exitCode}
            logs={action.logs}
            onRun={() => console.log('Run action')}
            onEdit={onEdit}
            onDelete={onDelete}
        />
    );
}

export function ActionItemCardMockup() {
    const [activeState, setActiveState] = useState<string>('basic');
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
    const [showEditModal, setShowEditModal] = useState(false);
    const [editIcon, setEditIcon] = useState('🔨');

    const states = [
        { id: 'basic', label: 'Basic' },
        { id: 'long-script', label: 'Long Script' },
        { id: 'running', label: 'Running' },
        { id: 'error', label: 'Error' },
        { id: 'success', label: 'Success' },
    ];

    const handleDelete = async () => {
        console.log('Action deleted');
        setShowDeleteConfirm(false);
    };

    const handleEdit = () => {
        setShowEditModal(true);
    };

    const handleSaveEdit = () => {
        console.log('Action edited');
        setShowEditModal(false);
    };

    const getActionName = (): string => {
        switch (activeState) {
            case 'basic': return 'Build Project';
            case 'long-script': return 'Run Tests';
            case 'running': return 'Deploy App';
            case 'error': return 'Lint Code';
            case 'success': return 'Format Code';
            default: return 'Unknown Action';
        }
    };

    return (
        <MockupPageContainer 
            title="Action Item Card Mockup" 
            description="Optimized for iPhone - Mobile-friendly action card design"
        >
            <div className="mockup-tabs">
                {states.map(state => (
                    <button
                        key={state.id}
                        className={`mockup-tab ${activeState === state.id ? 'active' : ''}`}
                        onClick={() => setActiveState(state.id)}
                    >
                        {state.label}
                    </button>
                ))}
            </div>

            <div className="mockup-preview">
                <ActionItemCard 
                    state={activeState as ActionItemCardProps['state']} 
                    onDelete={() => setShowDeleteConfirm(true)}
                    onEdit={handleEdit}
                />
            </div>

            {/* Delete Confirmation Modal */}
            {showDeleteConfirm && (
                <ConfirmModal
                    title="Delete Action"
                    message={`Are you sure you want to delete "${getActionName()}"? This action cannot be undone.`}
                    info={{
                        Action: getActionName(),
                    }}
                    command={`Delete action "${getActionName()}"`}
                    confirmLabel="Delete"
                    confirmVariant="danger"
                    onConfirm={handleDelete}
                    onClose={() => setShowDeleteConfirm(false)}
                />
            )}

            {/* Edit Modal */}
            {showEditModal && (
                <div className="mockup-edit-modal-overlay" onClick={() => setShowEditModal(false)}>
                    <div className="mockup-edit-modal" onClick={e => e.stopPropagation()}>
                        <div className="mockup-edit-modal-header">
                            <h3>Edit Action</h3>
                            <button className="mockup-edit-modal-close" onClick={() => setShowEditModal(false)}>×</button>
                        </div>
                        <div className="mockup-edit-modal-body">
                            <div className="mockup-edit-field">
                                <label>Name</label>
                                <input type="text" defaultValue={getActionName()} />
                            </div>
                            <div className="mockup-edit-field">
                                <label>Icon</label>
                                <ActionIconSelector
                                    value={editIcon}
                                    onChange={setEditIcon}
                                />
                            </div>
                            <div className="mockup-edit-field">
                                <label>Script</label>
                                <textarea rows={4} defaultValue="npm run build" />
                            </div>
                        </div>
                        <div className="mockup-edit-modal-footer">
                            <button className="mockup-edit-btn mockup-edit-btn-cancel" onClick={() => setShowEditModal(false)}>
                                Cancel
                            </button>
                            <button className="mockup-edit-btn mockup-edit-btn-save" onClick={handleSaveEdit}>
                                Save
                            </button>
                        </div>
                    </div>
                </div>
            )}

            <div className="mockup-notes">
                <h3>Design Notes:</h3>
                <ul>
                    <li><strong>Row 1:</strong> Icon + Name + Edit/Delete/Run buttons - clean horizontal layout</li>
                    <li><strong>Row 2:</strong> Script preview with max 3 lines, expandable</li>
                    <li><strong>Row 3:</strong> Logs section - collapsible, click to expand and stream</li>
                    <li><strong>States:</strong> Visual indicators for running (spinner), error (red), success (green)</li>
                    <li><strong>Touch targets:</strong> All buttons are minimum 44px for iOS accessibility</li>
                    <li><strong>Delete:</strong> Shows confirmation modal before deletion</li>
                </ul>
            </div>
        </MockupPageContainer>
    );
}
