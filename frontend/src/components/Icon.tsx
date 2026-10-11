import type { JSX } from 'react';

export type IconName =
    | 'info'
    | 'rss'
    | 'folder'
    | 'clock'
    | 'edit'
    | 'check'
    | 'backfill'
    | 'trash'
    | 'refresh'
    | 'chevron'
    | 'sun'
    | 'moon';
const paths: Record<IconName, JSX.Element> = {
    info: (
        <>
            <circle cx='12' cy='12' r='9' />
            <path d='M12 11v6M12 7v1' />
        </>
    ),
    rss: (
        <>
            <path d='M5 11a8 8 0 0 1 8 8M5 4a15 15 0 0 1 15 15' />
            <circle cx='5' cy='19' r='1' />
        </>
    ),
    folder: <path d='M3 7h7l2 2h9v11H3zM3 7V4h7l2 3' />,
    clock: (
        <>
            <circle cx='12' cy='12' r='9' />
            <path d='M12 7v5l3 2' />
        </>
    ),
    edit: <path d='m15 4 5 5-11 11H4v-5zM13 6l5 5' />,
    check: (
        <>
            <path d='m9 12 2 2 4-4' />
            <circle cx='12' cy='12' r='9' />
        </>
    ),
    backfill: <path d='M12 3v12m-4-4 4 4 4-4M4 15v5h16v-5' />,
    trash: <path d='M3 6h18M9 6V3h6v3M6 6l1 15h10l1-15M10 10v7m4-7v7' />,
    refresh: (
        <>
            <path d='M20 7v5h-5M4 17v-5h5' />
            <path d='M6 7a7 7 0 0 1 12-1l2 6M4 12l2 6a7 7 0 0 0 12-1' />
        </>
    ),
    chevron: <path d='m7 10 5 5 5-5' />,
    sun: (
        <>
            <circle cx='12' cy='12' r='4' />
            <path d='M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5' />
        </>
    ),
    moon: <path d='M20.5 13a8.5 8.5 0 0 1-9.5-9.5A8.5 8.5 0 1 0 20.5 13z' />,
};
export function Icon({ name }: { name: IconName }): JSX.Element {
    return (
        <svg
            aria-hidden='true'
            fill='none'
            stroke='currentColor'
            strokeLinecap='round'
            strokeLinejoin='round'
            strokeWidth='1.5'
            viewBox='0 0 24 24'
        >
            {paths[name]}
        </svg>
    );
}
