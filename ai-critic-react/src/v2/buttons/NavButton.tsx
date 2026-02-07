interface NavButtonProps {
    icon: React.ReactNode;
    label: string;
    active: boolean;
    onClick: () => void;
}

export function NavButton({ icon, label, active, onClick }: NavButtonProps) {
    return (
        <button className={`mcc-nav-btn ${active ? 'active' : ''}`} onClick={onClick}>
            {icon}
            <span>{label}</span>
        </button>
    );
}
