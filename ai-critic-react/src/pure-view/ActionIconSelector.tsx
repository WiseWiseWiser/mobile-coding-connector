import { useState, useRef, useEffect } from 'react';
import './ActionIconSelector.css';

export interface ActionIconOption {
    value: string;
    label: string;
}

export interface ActionIconGroup {
    name: string;
    options: ActionIconOption[];
}

export const ACTION_ICON_GROUPS: ActionIconGroup[] = [
    {
        name: 'Development',
        options: [
            { value: '🔨', label: 'Build' },
            { value: '▶️', label: 'Run' },
            { value: '🧪', label: 'Test' },
            { value: '📦', label: 'Package' },
            { value: '🔄', label: 'Update' },
            { value: '🧹', label: 'Clean' },
        ],
    },
    {
        name: 'Code Quality',
        options: [
            { value: '📋', label: 'Lint' },
            { value: '✨', label: 'Format' },
            { value: '🔍', label: 'Find' },
            { value: '✅', label: 'Verify' },
            { value: '🛡️', label: 'Security' },
        ],
    },
    {
        name: 'Deployment',
        options: [
            { value: '🚀', label: 'Deploy' },
            { value: '🌐', label: 'Web' },
            { value: '☁️', label: 'Cloud' },
            { value: '🔒', label: 'Secure' },
        ],
    },
    {
        name: 'Tools',
        options: [
            { value: '⚙️', label: 'Configure' },
            { value: '📊', label: 'Analyze' },
            { value: '💾', label: 'Save' },
            { value: '📁', label: 'Files' },
            { value: '🔗', label: 'Connect' },
            { value: '📝', label: 'Edit' },
        ],
    },
    {
        name: 'Status',
        options: [
            { value: '✅', label: 'Success' },
            { value: '❌', label: 'Error' },
            { value: '⚠️', label: 'Warning' },
            { value: 'ℹ️', label: 'Info' },
            { value: '🔄', label: 'Loading' },
            { value: '⏸️', label: 'Paused' },
        ],
    },
    {
        name: 'Misc',
        options: [
            { value: '🔔', label: 'Notify' },
            { value: '⭐', label: 'Favorite' },
            { value: '❤️', label: 'Love' },
            { value: '🎯', label: 'Goal' },
            { value: '💡', label: 'Idea' },
            { value: '🔧', label: 'Fix' },
        ],
    },
];

export const ACTION_ICON_OPTIONS: ActionIconOption[] = ACTION_ICON_GROUPS.flatMap(g => g.options);

export interface ActionIconSelectorProps {
    value: string;
    onChange: (value: string) => void;
    label?: string;
    className?: string;
}

export function ActionIconSelector({ value, onChange, label, className }: ActionIconSelectorProps) {
    const [isOpen, setIsOpen] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);

    const selectedOption = ACTION_ICON_OPTIONS.find(o => o.value === value);

    useEffect(() => {
        const handleClickOutside = (e: MouseEvent) => {
            if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
                setIsOpen(false);
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    return (
        <div className={`action-icon-selector ${className || ''}`} ref={containerRef}>
            {label && <label className="action-icon-selector-label">{label}</label>}
            <button
                type="button"
                className={`action-icon-selector-trigger ${isOpen ? 'open' : ''}`}
                onClick={() => setIsOpen(!isOpen)}
            >
                <span className="action-icon-selector-selected">{selectedOption?.value || '🔨'}</span>
                <span className="action-icon-selector-label-text">{selectedOption?.label || 'Select icon'}</span>
                <span className="action-icon-selector-arrow">{isOpen ? '▲' : '▼'}</span>
            </button>
            {isOpen && (
                <div className="action-icon-selector-dropdown">
                    {ACTION_ICON_GROUPS.map(group => (
                        <div key={group.name} className="action-icon-selector-group">
                            <div className="action-icon-selector-group-name">{group.name}</div>
                            <div className="action-icon-selector-group-options">
                                {group.options.map(option => (
                                    <button
                                        key={option.value}
                                        type="button"
                                        className={`action-icon-selector-option ${value === option.value ? 'selected' : ''}`}
                                        onClick={() => {
                                            onChange(option.value);
                                            setIsOpen(false);
                                        }}
                                    >
                                        <span className="action-icon-selector-option-icon">{option.value}</span>
                                        <span className="action-icon-selector-option-label">{option.label}</span>
                                    </button>
                                ))}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
