import './FormSelect.css';

export interface FormSelectProps {
    value: string;
    onChange: (value: string) => void;
    children: React.ReactNode;
    className?: string;
    disabled?: boolean;
}

export function FormSelect({ value, onChange, children, className, disabled }: FormSelectProps) {
    return (
        <select
            className={className ? `pure-form-select ${className}` : 'pure-form-select'}
            value={value}
            onChange={e => onChange(e.target.value)}
            disabled={disabled}
        >
            {children}
        </select>
    );
}
