import { useState, useRef, useEffect, useMemo } from 'react';
import './ModelSelector.css';

export interface ModelOption {
    id: string;
    name: string;
    providerId: string;
    providerName: string;
    is_default?: boolean;
    is_current?: boolean;
}

export interface ModelSelectorProps {
    models: ModelOption[];
    currentModel?: { modelID: string; providerID: string };
    onSelect: (model: { modelID: string; providerID: string }) => void;
    placeholder?: string;
    disabled?: boolean;
}

// Simple fuzzy search implementation
function fuzzyMatch(text: string, pattern: string): number {
    const textLower = text.toLowerCase();
    const patternLower = pattern.toLowerCase();
    
    // Exact match gets highest score
    if (textLower === patternLower) return 1000;
    
    // Starts with pattern gets high score
    if (textLower.startsWith(patternLower)) return 500;
    
    // Contains pattern gets medium score
    if (textLower.includes(patternLower)) return 100;
    
    // Fuzzy match - check if all characters in pattern appear in order
    let patternIdx = 0;
    let score = 0;
    let lastMatchIdx = -1;
    
    for (let i = 0; i < textLower.length && patternIdx < patternLower.length; i++) {
        if (textLower[i] === patternLower[patternIdx]) {
            // Bonus for consecutive matches
            if (lastMatchIdx === i - 1) {
                score += 10;
            } else {
                score += 1;
            }
            lastMatchIdx = i;
            patternIdx++;
        }
    }
    
    // If not all pattern characters matched, return 0
    if (patternIdx < patternLower.length) return 0;
    
    return score;
}

function filterModels(models: ModelOption[], query: string): ModelOption[] {
    if (!query.trim()) return models;
    
    const scored = models.map(model => {
        // Search in model name, model id, and provider name
        const nameScore = fuzzyMatch(model.name, query);
        const idScore = fuzzyMatch(model.id, query);
        const providerScore = fuzzyMatch(model.providerName, query);
        
        // Take the highest score
        const score = Math.max(nameScore, idScore, providerScore);
        
        return { model, score };
    });
    
    // Filter out non-matches and sort by score (descending)
    return scored
        .filter(item => item.score > 0)
        .sort((a, b) => b.score - a.score)
        .map(item => item.model);
}

export function ModelSelector({ 
    models, 
    currentModel, 
    onSelect, 
    placeholder = 'Select model...',
    disabled = false 
}: ModelSelectorProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [searchQuery, setSearchQuery] = useState('');
    const dropdownRef = useRef<HTMLDivElement>(null);
    const searchInputRef = useRef<HTMLInputElement>(null);

    // Close dropdown on outside click
    useEffect(() => {
        if (!isOpen) return;
        const handler = (e: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
                setIsOpen(false);
                setSearchQuery('');
            }
        };
        document.addEventListener('mousedown', handler);
        return () => document.removeEventListener('mousedown', handler);
    }, [isOpen]);

    // Focus search input when dropdown opens
    useEffect(() => {
        if (isOpen && searchInputRef.current) {
            searchInputRef.current.focus();
        }
    }, [isOpen]);

    const filteredModels = useMemo(() => {
        return filterModels(models, searchQuery);
    }, [models, searchQuery]);

    // Group filtered models by provider
    const groupedModels = useMemo(() => {
        const groups: Record<string, ModelOption[]> = {};
        filteredModels.forEach(model => {
            if (!groups[model.providerName]) {
                groups[model.providerName] = [];
            }
            groups[model.providerName].push(model);
        });
        return groups;
    }, [filteredModels]);

    // Find current model display info
    const currentModelInfo = useMemo(() => {
        if (!currentModel) return null;
        return models.find(m => 
            m.id === currentModel.modelID && m.providerId === currentModel.providerID
        );
    }, [models, currentModel]);

    const handleSelect = (model: ModelOption) => {
        onSelect({ modelID: model.id, providerID: model.providerId });
        setIsOpen(false);
        setSearchQuery('');
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape') {
            setIsOpen(false);
            setSearchQuery('');
        }
    };

    if (disabled || models.length === 0) {
        return (
            <span className="mcc-model-selector-trigger mcc-model-selector-disabled">
                {currentModelInfo ? currentModelInfo.name : placeholder}
            </span>
        );
    }

    return (
        <div className="mcc-model-selector" ref={dropdownRef}>
            <button 
                className="mcc-model-selector-trigger"
                onClick={() => setIsOpen(!isOpen)}
                type="button"
            >
                <span className="mcc-model-selector-current">
                    {currentModelInfo ? currentModelInfo.name : placeholder}
                </span>
                <span className="mcc-model-selector-chevron">▾</span>
            </button>

            {isOpen && (
                <div className="mcc-model-selector-dropdown">
                    <div className="mcc-model-selector-search">
                        <input
                            ref={searchInputRef}
                            type="text"
                            placeholder="Search models..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            onKeyDown={handleKeyDown}
                            className="mcc-model-selector-search-input"
                        />
                        {searchQuery && (
                            <span className="mcc-model-selector-search-count">
                                {filteredModels.length} found
                            </span>
                        )}
                    </div>

                    <div className="mcc-model-selector-list">
                        {filteredModels.length === 0 ? (
                            <div className="mcc-model-selector-empty">
                                No models match &quot;{searchQuery}&quot;
                            </div>
                        ) : (
                            Object.entries(groupedModels).map(([providerName, providerModels]) => (
                                <div key={providerName} className="mcc-model-selector-group">
                                    <div className="mcc-model-selector-provider">{providerName}</div>
                                    {providerModels.map(model => {
                                        const isActive = currentModel?.modelID === model.id && 
                                                        currentModel?.providerID === model.providerId;
                                        return (
                                            <button
                                                key={`${model.providerId}:${model.id}`}
                                                className={`mcc-model-selector-option${isActive ? ' mcc-model-selector-option-active' : ''}`}
                                                onClick={() => handleSelect(model)}
                                                type="button"
                                            >
                                                <span className="mcc-model-selector-option-name">{model.name}</span>
                                                <span className="mcc-model-selector-option-id">{model.id}</span>
                                                {model.is_default && (
                                                    <span className="mcc-model-selector-badge">default</span>
                                                )}
                                                {model.is_current && (
                                                    <span className="mcc-model-selector-badge mcc-model-selector-badge-current">current</span>
                                                )}
                                            </button>
                                        );
                                    })}
                                </div>
                            ))
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

export default ModelSelector;
