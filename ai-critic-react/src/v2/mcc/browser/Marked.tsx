import type { ReactNode } from 'react';
import { highlightSpans } from './tabSearch';

export interface MarkedProps {
    text: string;
    tokens: string[];
}

/**
 * Every occurrence of the query in `text`, wrapped in `<mark>` so the title and
 * the URL both show why they matched. `highlightSpans` merges overlaps, so the
 * marks never nest.
 */
export function Marked(props: MarkedProps) {
    const spans = highlightSpans(props.text, props.tokens);
    if (spans.length === 0) return <>{props.text}</>;
    const parts: ReactNode[] = [];
    let at = 0;
    spans.forEach((span, index) => {
        if (span.start > at) parts.push(props.text.slice(at, span.start));
        parts.push(
            <mark className="m" key={`m${index}`}>
                {props.text.slice(span.start, span.end)}
            </mark>,
        );
        at = span.end;
    });
    if (at < props.text.length) parts.push(props.text.slice(at));
    return <>{parts}</>;
}
