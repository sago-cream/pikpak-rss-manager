import { useEffect, useId, useRef } from 'react';
import type { JSX, ReactNode } from 'react';

import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';

export function Dialog({
    id,
    title,
    onClose,
    children,
    scroll = false,
    compact = false,
}: {
    id: string;
    title: string;
    onClose: () => void;
    children: ReactNode;
    scroll?: boolean;
    compact?: boolean;
}): JSX.Element {
    const ref = useRef<HTMLDialogElement>(null);
    const titleId = useId();
    const { t } = usePreferences();
    useEffect(() => {
        const dialog = ref.current;
        if (dialog && !dialog.open) {
            dialog.showModal();
        }
    }, []);
    const content = (
        <>
            <div className='dialog-heading'>
                <h2 id={titleId}>{title}</h2>
                <button
                    aria-label={t('關閉')}
                    className='icon-button'
                    onClick={onClose}
                    type='button'
                >
                    <Icon name='close' />
                </button>
            </div>
            {children}
        </>
    );
    return (
        <dialog
            aria-labelledby={titleId}
            className={compact ? 'compact-dialog' : undefined}
            id={id}
            onCancel={(event) => {
                event.preventDefault();
                onClose();
            }}
            onClose={onClose}
            ref={ref}
        >
            {scroll ? <div className='dialog-scroll'>{content}</div> : content}
        </dialog>
    );
}
