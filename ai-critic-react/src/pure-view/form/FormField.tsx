import './FormField.css';

export interface FormFieldProps {
    label: string;
    children: React.ReactNode;
    className?: string;
}

export function FormField({ label, children, className }: FormFieldProps) {
    return (
        <div className={className ? `pure-form-field ${className}` : 'pure-form-field'}>
            <label className="pure-form-field-label">{label}</label>
            {children}
        </div>
    );
}
