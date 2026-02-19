import { FlexInput, type FlexInputProps } from './FlexInput';
import './PathInput.css';

export interface PathInputProps extends Omit<FlexInputProps, 'multiline'> {
    label?: string;
}

export function PathInput({ value, onChange, label = 'Project Directory', ...props }: PathInputProps) {
    return (
        <div className="path-input-container">
            {label && <h3 className="path-input-label">{label}</h3>}
            <FlexInput
                value={value}
                onChange={onChange}
                multiline
                rows={1}
                spellCheck={false}
                {...props}
            />
        </div>
    );
}
