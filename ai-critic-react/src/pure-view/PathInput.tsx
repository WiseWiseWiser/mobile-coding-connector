import './PathInput.css';

export interface PathInputProps {
    value: string;
    onChange: (value: string) => void;
    label?: string;
}

export function PathInput({ value, onChange, label = 'Project Directory' }: PathInputProps) {
    return (
        <div className="path-input-container">
            {label && <h3 className="path-input-label">{label}</h3>}
            <input
                type="text"
                className="path-input-field"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                spellCheck={false}
            />
        </div>
    );
}
