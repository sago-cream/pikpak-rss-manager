import type { JSX } from 'react';

export function ProgressMeter({
    value,
    label,
    valueText,
    kind = 'progressbar',
    segmented = false,
    warning = false,
}: {
    value: number;
    label: string;
    valueText?: string;
    kind?: 'progressbar' | 'meter';
    segmented?: boolean;
    warning?: boolean;
}): JSX.Element {
    const percent = Math.min(100, Math.max(0, value));
    return (
        <div
            aria-label={label}
            aria-valuemax={100}
            aria-valuemin={0}
            aria-valuenow={percent}
            aria-valuetext={valueText}
            className={`progress-meter${warning ? ' is-warning' : ''}`}
            role={kind}
        >
            <svg
                aria-hidden='true'
                preserveAspectRatio='none'
                viewBox='0 0 100 4'
            >
                <rect className='meter-track' height='4' width='100' />
                <rect className='meter-fill' height='4' width={percent} />
                {segmented && (
                    <path
                        className='meter-segments'
                        d={Array.from(
                            { length: 19 },
                            (_, i) => `M${(i + 1) * 5} 0v4`
                        ).join(' ')}
                    />
                )}
            </svg>
        </div>
    );
}
