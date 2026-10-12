import type { JSX, ReactNode } from 'react';

export function WorkspaceRow({
    kind,
    id,
    title,
    label,
    onOpen,
    leading,
    trailing,
    paused,
    disabled = false,
    children,
}: {
    kind: 'subscription' | 'job';
    id: number | string;
    title: string;
    label: string;
    onOpen: () => void;
    leading: ReactNode;
    trailing?: ReactNode;
    paused?: boolean;
    disabled?: boolean;
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
                        disabled={disabled}
                        onClick={onOpen}
                        type='button'
                    >
                        {title}
                    </button>
                </h3>
                {children}
            </div>
            {trailing}
        </article>
    );
}
