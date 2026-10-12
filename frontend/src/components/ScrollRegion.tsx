import type { JSX, ReactNode } from 'react';

export function ScrollRegion({
    id,
    label,
    children,
}: {
    id: string;
    label: string;
    children: ReactNode;
}): JSX.Element {
    return (
        <div
            aria-label={label}
            className='workspace-scroll'
            id={id}
            role='group'
            tabIndex={0}
        >
            {children}
        </div>
    );
}
