import type { JSX, ReactNode } from 'react';

import { Icon } from './Icon.js';

export function WorkspaceRow({
    kind,
    id,
    title,
    label,
    onOpen,
    leading,
    paused,
    children,
}: {
    kind: 'subscription' | 'job';
    id: number | string;
    title: string;
    label: string;
    onOpen: () => void;
    leading: ReactNode;
    paused?: boolean;
    children: ReactNode;
}): JSX.Element {
    return (
        <article className={`workspace-row ${kind}-row`} data-paused={paused}>
            {leading}
            <div className={`workspace-row-info ${kind}-info`}>
                <h3>
                    <button
                        aria-label={label}
                        className='workspace-row-open'
                        data-edit={kind === 'subscription' ? id : undefined}
                        data-job={kind === 'job' ? id : undefined}
                        onClick={onOpen}
                        type='button'
                    >
                        {title}
                    </button>
                </h3>
                {children}
            </div>
            <button
                aria-label={label}
                className={`icon-button workspace-row-menu ${kind}-menu`}
                data-subscription-settings={
                    kind === 'subscription' ? id : undefined
                }
                data-task-details={kind === 'job' ? id : undefined}
                onClick={onOpen}
                title={label}
                type='button'
            >
                <Icon name='more-vertical' />
            </button>
        </article>
    );
}
