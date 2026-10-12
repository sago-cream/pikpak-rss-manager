import type { JSX, ReactNode } from 'react';

import { ScrollRegion } from './ScrollRegion.js';

export function WorkspacePanel({
    id,
    titleId,
    title,
    controls,
    actions,
    listId,
    listLabel,
    children,
}: {
    id: string;
    titleId: string;
    title: string;
    controls?: ReactNode;
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
                {controls && (
                    <div className='workspace-section-controls'>{controls}</div>
                )}
                <div className='section-actions'>{actions}</div>
            </header>
            <ScrollRegion id={listId} label={listLabel}>
                {children}
            </ScrollRegion>
        </section>
    );
}
