import { formatBytes } from '../../../api/processes';

export interface ChartPoint {
    /** Short x-axis label, e.g. "11-05". */
    label: string;
    /** `null` leaves a gap: the report is missing or measured differently. */
    value: number | null;
}

interface Props {
    points: ChartPoint[];
    color?: string;
    height?: number;
    /** Index drawn with a bigger dot (usually the newest or the inspected run). */
    highlight?: number;
}

const WIDTH = 320;
const PAD_LEFT = 38;
const PAD_RIGHT = 8;
const PAD_TOP = 10;
const PAD_BOTTOM = 22;

function compact(bytes: number): string {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let n = Math.abs(bytes);
    let i = 0;
    while (n >= 1024 && i < units.length - 1) {
        n /= 1024;
        i++;
    }
    const digits = n >= 100 || i === 0 ? 0 : n >= 10 ? 0 : 1;
    return `${n.toFixed(digits)}${units[i]}`;
}

function niceMax(value: number): number {
    if (value <= 0) return 1;
    const magnitude = Math.pow(10, Math.floor(Math.log10(value)));
    const scaled = value / magnitude;
    const step = scaled <= 1 ? 1 : scaled <= 2 ? 2 : scaled <= 5 ? 5 : 10;
    return step * magnitude;
}

/** Line chart with real gaps, drawn in SVG so it needs no chart library. */
export default function SpaceLineChart({ points, color = '#22d3ee', height = 128, highlight = -1 }: Props) {
    const known = points.filter(p => typeof p.value === 'number' && Number.isFinite(p.value as number));
    if (known.length === 0) {
        return <div className="mcc-system-empty">Not enough reports to draw a curve yet</div>;
    }

    const plotBottom = height - PAD_BOTTOM;
    const plotHeight = plotBottom - PAD_TOP;
    const plotWidth = WIDTH - PAD_LEFT - PAD_RIGHT;
    const peak = Math.max(...known.map(p => p.value as number));
    const max = niceMax(peak);
    const step = points.length > 1 ? plotWidth / (points.length - 1) : 0;
    const project = (v: number) => plotBottom - (v / max) * plotHeight;
    const xAt = (index: number) => PAD_LEFT + index * step;

    const segments: { points: string; last: { x: number; y: number } | null }[] = [];
    let current: string[] = [];
    let tail: { x: number; y: number } | null = null;
    points.forEach((point, index) => {
        if (typeof point.value !== 'number' || !Number.isFinite(point.value)) {
            if (current.length > 1) segments.push({ points: current.join(' '), last: tail });
            current = [];
            tail = null;
            return;
        }
        const x = xAt(index);
        const y = project(point.value);
        current.push(`${x.toFixed(1)},${y.toFixed(1)}`);
        tail = { x, y };
    });
    if (current.length > 1) segments.push({ points: current.join(' '), last: tail });
    else if (current.length === 1) segments.push({ points: '', last: tail });

    const areaSegments = segments
        .filter(segment => segment.points && segment.last)
        .map(segment => {
            const coords = segment.points.split(' ');
            const firstX = coords[0].split(',')[0];
            const lastX = coords[coords.length - 1].split(',')[0];
            return `${segment.points} ${lastX},${plotBottom} ${firstX},${plotBottom}`;
        });

    const gridValues = [1, 0.66, 0.33, 0];
    const labels = labelIndexes(points.length);
    const lastKnownIndex = points.reduce((acc, p, i) => (typeof p.value === 'number' ? i : acc), -1);
    const markIndex = highlight >= 0 && highlight < points.length ? highlight : lastKnownIndex;
    const mark = markIndex >= 0 && typeof points[markIndex]?.value === 'number'
        ? { x: xAt(markIndex), y: project(points[markIndex].value as number) }
        : null;

    return (
        <div className="mcc-space-chart">
            <svg width="100%" viewBox={`0 0 ${WIDTH} ${height}`} preserveAspectRatio="none" role="img">
                {gridValues.map(ratio => {
                    const y = PAD_TOP + plotHeight * (1 - ratio);
                    return (
                        <g key={ratio}>
                            <line
                                x1={PAD_LEFT}
                                y1={y}
                                x2={WIDTH - PAD_RIGHT}
                                y2={y}
                                stroke="#1f2b3d"
                                strokeWidth={1}
                                strokeDasharray="2 3"
                            />
                            <text x={2} y={y + 3} fill="#64748b" fontSize={8}>
                                {compact(max * ratio)}
                            </text>
                        </g>
                    );
                })}
                {areaSegments.map(area => (
                    <polygon key={area} points={area} fill={color} opacity={0.1} />
                ))}
                {segments.map((segment, index) =>
                    segment.points ? (
                        <polyline
                            key={`line-${index}`}
                            points={segment.points}
                            fill="none"
                            stroke={color}
                            strokeWidth={2}
                            strokeLinecap="round"
                            strokeLinejoin="round"
                        />
                    ) : null
                )}
                {segments.map((segment, index) =>
                    segment.points ? null : segment.last ? (
                        <circle key={`dot-${index}`} cx={segment.last.x} cy={segment.last.y} r={2.4} fill={color} />
                    ) : null
                )}
                {mark && <circle cx={mark.x} cy={mark.y} r={3.4} fill={color} stroke="#0f172a" strokeWidth={1.5} />}
            </svg>
            <div className="mcc-space-xlab">
                {points.map((point, index) => (
                    <span key={`${point.label}-${index}`}>{labels.has(index) ? point.label : ''}</span>
                ))}
            </div>
            <div className="mcc-space-axis-note">
                peak {formatBytes(peak)} · {points.length} {points.length === 1 ? 'report' : 'reports'}
            </div>
        </div>
    );
}

/** Keep at most five x labels so narrow phones do not collide. */
function labelIndexes(count: number): Set<number> {
    if (count <= 5) {
        return new Set(Array.from({ length: count }, (_, i) => i));
    }
    const wanted = new Set<number>([0, count - 1]);
    const slots = 4;
    for (let i = 1; i < slots; i++) {
        wanted.add(Math.round((i * (count - 1)) / slots));
    }
    return wanted;
}
