export function statusBadge(status: string) {
    const cls = `mcc-file-status mcc-file-status-${status}`;
    const label = status === 'added' ? 'A' : status === 'deleted' ? 'D' : 'M';
    return <span className={cls}>{label}</span>;
}
