import {
    useCallback,
    useEffect,
    useId,
    useMemo,
    useRef,
    useState,
} from 'react';
import type { JSX, ReactNode } from 'react';

import { DialogFormContext } from './DirtyForm.js';
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
    const [opener] = useState(() =>
        document.activeElement instanceof HTMLElement
            ? document.activeElement
            : null
    );
    const dirtyForms = useRef(new Set<string>());
    const titleId = useId();
    const { t } = usePreferences();
    const register = useCallback((key: string, dirty: boolean) => {
        if (dirty) {
            dirtyForms.current.add(key);
        } else {
            dirtyForms.current.delete(key);
        }
    }, []);
    const requestClose = useCallback(() => {
        if (
            dirtyForms.current.size > 0 &&
            !globalThis.confirm(t('放棄未儲存的變更？'))
        ) {
            return;
        }
        onClose();
    }, [onClose, t]);
    const scope = useMemo(
        () => ({ register, requestClose }),
        [register, requestClose]
    );
    useEffect(() => {
        const dialog = ref.current;
        if (dialog) {
            if (!dialog.open) {
                dialog.showModal();
            }
            return () => {
                globalThis.requestAnimationFrame(() => {
                    const topDialog = [
                        ...document.querySelectorAll('dialog[open]'),
                    ].at(-1);
                    if (topDialog && (!opener || !topDialog.contains(opener))) {
                        return;
                    }
                    const target =
                        opener?.isConnected &&
                        opener.getClientRects().length > 0 &&
                        !opener.closest('[popover]')
                            ? opener
                            : document.querySelector<HTMLElement>(
                                  '#profile-toggle'
                              );
                    target?.focus({ preventScroll: true });
                });
            };
        }
    }, [opener]);
    const content = (
        <>
            <div className='dialog-heading'>
                <h2 id={titleId}>{title}</h2>
                <button
                    aria-label={t('關閉')}
                    className='icon-button'
                    onClick={requestClose}
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
                if (event.target !== event.currentTarget) {
                    return;
                }
                event.preventDefault();
                event.stopPropagation();
                requestClose();
            }}
            onClose={(event) => {
                if (event.target !== event.currentTarget) {
                    return;
                }
                event.stopPropagation();
                requestClose();
            }}
            ref={ref}
        >
            <DialogFormContext.Provider value={scope}>
                {scroll ? (
                    <div className='dialog-scroll'>{content}</div>
                ) : (
                    content
                )}
            </DialogFormContext.Provider>
        </dialog>
    );
}
