import { useCallback, useEffect, useRef, useState } from 'react';
import type { JSX } from 'react';

import { api } from '../lib/api.js';
import { errorMessage, isCancelled } from '../lib/i18n.js';
import type { FolderListing } from '../lib/types.js';
import { Dialog } from './Dialog.js';
import { DialogCloseButton, DirtyForm } from './DirtyForm.js';
import { Feedback, FormField } from './FormField.js';
import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';

export interface Destination {
    path: string;
    id: string;
    account: string;
}
export function FolderPicker({
    initialID,
    onSelect,
    onClose,
}: {
    initialID: string;
    onSelect: (value: Destination) => void;
    onClose: () => void;
}): JSX.Element {
    const { t } = usePreferences();
    const [creating, setCreating] = useState(false);
    const [error, setError] = useState('');
    const [listing, setListing] = useState<FolderListing | null>(null);
    const [pending, setPending] = useState(false);
    const [search, setSearch] = useState('');
    const request = useRef<AbortController | null>(null);
    const load = useCallback(
        async (parent = '', token = '', appendTo?: FolderListing) => {
            request.current?.abort();
            const controller = new AbortController();
            request.current = controller;
            setPending(true);
            setError('');
            if (!appendTo) {
                setListing(null);
                setSearch('');
            }
            try {
                const result = await api<FolderListing>(
                    `/api/pikpak/folders?${new URLSearchParams({ parent_id: parent, token })}`,
                    'GET',
                    undefined,
                    controller.signal
                );
                if (controller.signal.aborted) {
                    return;
                }
                if (appendTo) {
                    if (appendTo.account_ref !== result.account_ref) {
                        throw new Error(
                            'PikPak 帳號已更換，請重新開啟資料夾選擇器'
                        );
                    }
                    const known = new Set(
                        appendTo.folders.map((folder) => folder.id)
                    );
                    result.folders = [
                        ...appendTo.folders,
                        ...result.folders.filter(
                            (folder) => !known.has(folder.id)
                        ),
                    ];
                }
                setListing(result);
            } catch (error_) {
                if (!controller.signal.aborted && !isCancelled(error_)) {
                    setError(errorMessage(error_));
                }
            } finally {
                if (!controller.signal.aborted) {
                    setPending(false);
                }
            }
        },
        []
    );
    useEffect(() => {
        // The API response updates state asynchronously; cleanup cancels the request.
        // eslint-disable-next-line react-hooks/set-state-in-effect
        void load(initialID);
        return () =>
            request.current?.abort(); /* A picker owns one initial selection until it is closed. */
    }, [initialID, load]);
    return (
        <div className='folder-browser full' id='folder-browser'>
            <div className='folder-heading'>
                <nav
                    aria-label={t('資料夾路徑')}
                    className='folder-breadcrumbs'
                    id='folder-breadcrumbs'
                >
                    {listing?.breadcrumbs.map((folder, index) => (
                        <span key={folder.id}>
                            {index > 0 && ' / '}
                            <button
                                aria-busy={pending}
                                className='text-button'
                                data-folder={folder.id}
                                disabled={pending}
                                onClick={() => {
                                    void load(folder.id);
                                }}
                                type='button'
                            >
                                {folder.id ? folder.name : t('根目錄')}
                            </button>
                        </span>
                    ))}
                </nav>
                <button
                    aria-label={t('收起資料夾選擇器')}
                    className='icon-button'
                    id='folder-close'
                    onClick={onClose}
                    type='button'
                >
                    <Icon name='close' />
                </button>
            </div>
            {listing && (listing.folders.length > 8 || listing.next_token) && (
                <input
                    aria-label={t('搜尋目前資料夾')}
                    id='folder-search'
                    onChange={(event) => {
                        setSearch(event.target.value);
                    }}
                    placeholder={t('搜尋目前已載入的資料夾')}
                    type='search'
                    value={search}
                />
            )}
            <p
                className={`field-help${error ? ' error' : ''}`}
                id='folder-status'
                role='status'
            >
                {pending ? t('正在讀取 PikPak 資料夾…') : t(error)}
            </p>
            <div className='folder-list' id='folder-list'>
                {listing?.folders
                    .filter((folder) =>
                        folder.name.toLowerCase().includes(search.toLowerCase())
                    )
                    .map((folder) => (
                        <button
                            aria-busy={pending}
                            className='folder-row'
                            data-folder={folder.id}
                            disabled={pending}
                            key={folder.id}
                            onClick={() => {
                                void load(folder.id);
                            }}
                            type='button'
                        >
                            <Icon className='folder-icon' name='folder' />
                            <span>{folder.name}</span>
                            <Icon
                                className='folder-arrow'
                                name='chevron-right'
                            />
                        </button>
                    ))}
                {listing &&
                    !listing.folders.some((folder) =>
                        folder.name.toLowerCase().includes(search.toLowerCase())
                    ) && (
                        <p className='field-help'>
                            {t('目前沒有符合的資料夾。')}
                        </p>
                    )}
            </div>
            {listing?.next_token && (
                <button
                    className='text-button'
                    disabled={pending}
                    id='folder-more'
                    onClick={() => {
                        void load(
                            listing.current.id,
                            listing.next_token,
                            listing
                        );
                    }}
                    type='button'
                >
                    {t('載入更多資料夾')}
                </button>
            )}
            <div className='folder-actions'>
                <button
                    className='text-button'
                    disabled={!listing || pending || Boolean(error)}
                    id='folder-create'
                    onClick={() => {
                        setCreating(true);
                    }}
                    type='button'
                >
                    <Icon name='plus' />
                    {t('新增資料夾')}
                </button>
                <button
                    className='button primary'
                    disabled={!listing || pending || Boolean(error)}
                    id='folder-select'
                    onClick={() => {
                        if (listing) {
                            onSelect({
                                path: listing.current.path,
                                id: listing.current.id,
                                account: listing.account_ref,
                            });
                        }
                    }}
                    type='button'
                >
                    {t(listing?.current.id ? '使用此資料夾' : '使用根目錄')}
                </button>
            </div>
            {creating && listing && (
                <CreateFolder
                    listing={listing}
                    onClose={() => {
                        setCreating(false);
                    }}
                    onCreated={async (id) => {
                        setCreating(false);
                        await load(id);
                    }}
                />
            )}
        </div>
    );
}
function CreateFolder({
    listing,
    onClose,
    onCreated,
}: {
    listing: FolderListing;
    onClose: () => void;
    onCreated: (id: string) => Promise<void>;
}): JSX.Element {
    const { t } = usePreferences();
    const [error, setError] = useState('');
    const [name, setName] = useState('');
    const [pending, setPending] = useState(false);
    const mounted = useRef(true);
    useEffect(() => {
        mounted.current = true;
        return () => {
            mounted.current = false;
        };
    }, []);
    return (
        <Dialog
            compact
            id='folder-create-dialog'
            onClose={onClose}
            title={t('新增資料夾')}
        >
            <form
                id='folder-create-form'
                onSubmit={(event) => {
                    event.preventDefault();
                    if (pending) {
                        return;
                    }
                    setPending(true);
                    setError('');
                    void api<{ folder: { id: string } }>(
                        '/api/pikpak/folders',
                        'POST',
                        {
                            parent_id: listing.current.id,
                            account_ref: listing.account_ref,
                            name,
                        }
                    )
                        .then(async (result) => {
                            if (mounted.current) {
                                await onCreated(result.folder.id);
                            }
                        })
                        .catch((error_: unknown) => {
                            if (mounted.current) {
                                setError(errorMessage(error_));
                            }
                        })
                        .finally(() => {
                            if (mounted.current) {
                                setPending(false);
                            }
                        });
                }}
            >
                <DirtyForm dirty={Boolean(name || pending)} />
                <p className='field-help' id='folder-create-location'>
                    {t('建立位置：')}
                    {listing.current.path || t('根目錄')}
                </p>
                <FormField id='new-folder-name' label={t('資料夾名稱')}>
                    <input
                        autoFocus
                        maxLength={100}
                        onChange={(event) => {
                            setName(event.target.value);
                        }}
                        placeholder={t('請輸入新資料夾名稱')}
                        required
                        value={name}
                    />
                </FormField>
                <Feedback message={t(error)} />
                <div className='dialog-footer'>
                    <DialogCloseButton className='button subtle'>
                        {t('取消')}
                    </DialogCloseButton>
                    <button
                        aria-busy={pending}
                        className='button primary'
                        disabled={pending || !name.trim()}
                        type='submit'
                    >
                        {pending && (
                            <Icon className='is-loading' name='loading' />
                        )}
                        {t('建立資料夾')}
                    </button>
                </div>
            </form>
        </Dialog>
    );
}
