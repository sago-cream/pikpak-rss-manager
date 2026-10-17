import type { JSX, ReactNode } from 'react';

import { ScrollRegion } from './ScrollRegion.js';

export function WorkspacePanel({
    id,
    titleId,
    title,
    actions,
    listId,
    listLabel,
    children,
}: {
    id: string;
    titleId: string;
    title: string;
    actions: ReactNode;
    listId: string;
    listLabel: string;
    children: ReactNode;
}): JSX.Element {
    return (
        <section
            aria-labelledby={titleId}
            className='workspace-section'
            id={id}
        >
            <header className='workspace-section-heading'>
                <h2 id={titleId}>{title}</h2>
                <div className='section-actions'>{actions}</div>
            </header>
            <ScrollRegion id={listId} label={listLabel}>
                {children}
            </ScrollRegion>
        </section>
    );
}
