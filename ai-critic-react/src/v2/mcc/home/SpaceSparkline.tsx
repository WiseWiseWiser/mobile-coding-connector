interface Props {
    /** Oldest first. `null` breaks the line instead of drawing a fake zero. */
    values: (number | null)[];
    color?: string;
    width?: number;
    height?: number;
}

/** Tiny trend line for the entry card and history rows. */
export default function SpaceSparkline({ values, color = '#22d3ee', width = 64, height = 26 }: Props) {
    const known = values.filter((v): v is number => typeof v === 'number' && Number.isFinite(v));
    if (known.length < 2) return null;

    const min = Math.min(...known);
    const max = Math.max(...known);
    const span = max - min || 1;
    const pad = 3;
    const step = values.length > 1 ? (width - pad * 2) / (values.length - 1) : 0;
    const project = (v: number) => height - pad - ((v - min) / span) * (height - pad * 2);

    const segments: string[] = [];
    let current: string[] = [];
    let lastX = pad;
    let lastY = height / 2;
    values.forEach((value, index) => {
        if (typeof value !== 'number' || !Number.isFinite(value)) {
            if (current.length > 1) segments.push(current.join(' '));
            current = [];
            return;
        }
        const x = pad + index * step;
        const y = project(value);
        current.push(`${x.toFixed(1)},${y.toFixed(1)}`);
        lastX = x;
        lastY = y;
    });
    if (current.length > 1) segments.push(current.join(' '));
    if (segments.length === 0) return null;

    return (
        <svg className="mcc-space-spark" width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
            {segments.map(points => (
                <polyline
                    key={points}
                    points={points}
                    fill="none"
                    stroke={color}
                    strokeWidth={1.6}
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />
            ))}
            <circle cx={lastX} cy={lastY} r={2} fill={color} />
        </svg>
    );
}
