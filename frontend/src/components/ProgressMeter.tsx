import type { JSX } from 'react';

export function ProgressMeter({
    value,
    label,
    valueText,
    kind = 'progressbar',
    segmented = false,
    warning = false,
    shape = 'line',
}: {
    value: number;
    label: string;
    valueText?: string;
    kind?: 'progressbar' | 'meter';
    segmented?: boolean;
    warning?: boolean;
    shape?: 'line' | 'circle';
}): JSX.Element {
    const percent = Math.min(100, Math.max(0, value));
    return (
        <span
            aria-label={label}
            aria-valuemax={100}
            aria-valuemin={0}
            aria-valuenow={percent}
            aria-valuetext={valueText}
            className={`progress-meter${shape === 'circle' ? ' is-circular' : ''}${warning ? ' is-warning' : ''}`}
            role={kind}
        >
            {shape === 'circle' ? (
                <svg aria-hidden='true' viewBox='0 0 24 24'>
                    <circle className='meter-track' cx='12' cy='12' r='9' />
                    {percent > 0 && (
                        <circle
                            className='meter-fill'
                            cx='12'
                            cy='12'
                            pathLength='100'
                            r='9'
                            strokeDasharray={`${percent} 100`}
                            transform='rotate(-90 12 12)'
                        />
                    )}
                </svg>
            ) : (
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
            )}
        </span>
    );
}
