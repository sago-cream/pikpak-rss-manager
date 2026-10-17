import {
    fireEvent,
    render,
    screen,
    waitFor,
    within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { Preferences } from '../src/components/Preferences.js';
import { SubscriptionDialog } from '../src/components/SubscriptionDialog.js';

const success = async () => undefined;
const requestPath = (input: RequestInfo | URL): string =>
    typeof input === 'string'
        ? input
        : input instanceof URL
          ? input.href
          : input.url;
const listing = (id = '') => ({
    current: {
        id,
        name: id ? 'Fixture folder' : '',
        path: id ? 'Fixture folder' : '',
    },
    breadcrumbs: [{ id: '', name: '', path: '' }],
    folders: [],
    account_ref: 'fixture-account',
});

function renderDraft(
    manual: boolean,
    onSaved: () => Promise<void> = success,
    onRefreshError?: (error: unknown) => void
) {
    const onClose = vi.fn<() => void>();
    const view = render(
        <Preferences>
            <SubscriptionDialog
                manual={manual}
                onClose={onClose}
                onRefreshError={onRefreshError}
                onSaved={onSaved}
                status={{ connected: true }}
            />
        </Preferences>
    );
    if (!manual) {
        fireEvent.change(screen.getByLabelText('訂閱名稱'), {
            target: { value: 'Fixture subscription' },
        });
    }
    fireEvent.change(screen.getByLabelText(manual ? '下載連結' : 'RSS 連結'), {
        target: { value: 'https://fixture.test/source' },
    });
    return { ...view, onClose };
}

async function openFolderDialog() {
    await userEvent.click(screen.getByRole('button', { name: '選擇資料夾' }));
    await userEvent.click(
        await screen.findByRole('button', { name: '新增資料夾' })
    );
    return screen.getByRole('dialog', { name: '新增資料夾' });
}

describe('Dialog submission boundaries', () => {
    it.each([true, false])(
        'dismisses only the active folder dialog and preserves its parent draft (manual=%s)',
        async (manual) => {
            const fetch = vi
                .spyOn(globalThis, 'fetch')
                .mockResolvedValue(Response.json(listing()));
            const confirm = vi
                .spyOn(globalThis, 'confirm')
                .mockReturnValue(true);
            const { onClose } = renderDraft(manual);
            const child = await openFolderDialog();
            const cancel = new Event('cancel', { cancelable: true });
            fireEvent(child, cancel);
            expect(cancel.defaultPrevented).toBe(true);
            expect(
                screen.queryByRole('dialog', { name: '新增資料夾' })
            ).toBeNull();
            expect(onClose).not.toHaveBeenCalled();
            expect(confirm).not.toHaveBeenCalled();
            expect(
                screen.getByLabelText(manual ? '下載連結' : 'RSS 連結')
            ).toHaveProperty('value', 'https://fixture.test/source');
            if (!manual) {
                expect(screen.getByLabelText('訂閱名稱')).toHaveProperty(
                    'value',
                    'Fixture subscription'
                );
            }
            expect(
                document.querySelector('#subscription-dialog')
            ).toHaveProperty('open', true);
            expect(
                fetch.mock.calls.every(
                    ([, options]) => options?.method === 'GET'
                )
            ).toBe(true);
            fireEvent(
                document.querySelector('#subscription-dialog')!,
                new Event('cancel', { cancelable: true })
            );
            expect(confirm).toHaveBeenCalledOnce();
            expect(onClose).toHaveBeenCalledOnce();
        }
    );
    it('keeps a changed child draft when discard is declined and closes only that child when accepted', async () => {
        vi.spyOn(globalThis, 'fetch').mockResolvedValue(
            Response.json(listing())
        );
        const confirm = vi.spyOn(globalThis, 'confirm').mockReturnValue(false);
        const { onClose } = renderDraft(true);
        const child = await openFolderDialog();
        fireEvent.change(within(child).getByLabelText('資料夾名稱'), {
            target: { value: 'Fixture child draft' },
        });
        fireEvent(child, new Event('cancel', { cancelable: true }));
        expect(confirm).toHaveBeenCalledOnce();
        expect(child).toHaveProperty('open', true);
        expect(within(child).getByLabelText('資料夾名稱')).toHaveProperty(
            'value',
            'Fixture child draft'
        );
        expect(onClose).not.toHaveBeenCalled();
        confirm.mockReturnValue(true);
        fireEvent(child, new Event('cancel', { cancelable: true }));
        expect(confirm).toHaveBeenCalledTimes(2);
        expect(screen.queryByRole('dialog', { name: '新增資料夾' })).toBeNull();
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.getByLabelText('下載連結')).toHaveProperty(
            'value',
            'https://fixture.test/source'
        );
    });
    it('scopes a native close event to the active child dialog', async () => {
        vi.spyOn(globalThis, 'fetch').mockResolvedValue(
            Response.json(listing())
        );
        const confirm = vi.spyOn(globalThis, 'confirm').mockReturnValue(true);
        const { onClose } = renderDraft(true);
        const child = await openFolderDialog();
        fireEvent(child, new Event('close'));
        expect(screen.queryByRole('dialog', { name: '新增資料夾' })).toBeNull();
        expect(confirm).not.toHaveBeenCalled();
        expect(onClose).not.toHaveBeenCalled();
    });
    it.each([true, false])(
        'finishes an accepted create before a failed refresh (manual=%s)',
        async (manual) => {
            const fetch = vi
                .spyOn(globalThis, 'fetch')
                .mockResolvedValue(Response.json({ ok: true }));
            const refresh = Promise.withResolvers<undefined>();
            const onSaved = vi.fn(async () => {
                await refresh.promise;
            });
            const onRefreshError = vi.fn<(error: unknown) => void>();
            const { onClose } = renderDraft(manual, onSaved, onRefreshError);
            const form = document.querySelector('#subscription-form')!;
            fireEvent.submit(form);
            await waitFor(() => {
                expect(onSaved).toHaveBeenCalledOnce();
            });
            expect(onClose).toHaveBeenCalledOnce();
            expect(
                screen.getByRole('button', {
                    name: manual ? '新增任務' : '儲存訂閱',
                })
            ).toHaveProperty('disabled', true);
            fireEvent.submit(form);
            const error = new Error('Fixture refresh failed');
            refresh.reject(error);
            await waitFor(() => {
                expect(onRefreshError).toHaveBeenCalledWith(error);
            });
            fireEvent.submit(form);
            expect(fetch).toHaveBeenCalledOnce();
            expect(onClose).toHaveBeenCalledOnce();
            expect(screen.queryByRole('alert')).toBeNull();
        }
    );
    it.each([true, false])(
        'allows retrying a rejected create without refreshing (manual=%s)',
        async (manual) => {
            const fetch = vi
                .spyOn(globalThis, 'fetch')
                .mockResolvedValueOnce(
                    Response.json(
                        { error: 'Fixture create failed' },
                        { status: 400 }
                    )
                )
                .mockResolvedValueOnce(Response.json({ ok: true }));
            const onSaved = vi.fn(success);
            const { onClose } = renderDraft(manual, onSaved);
            fireEvent.submit(document.querySelector('#subscription-form')!);
            await screen.findByRole('alert');
            expect(onSaved).not.toHaveBeenCalled();
            expect(onClose).not.toHaveBeenCalled();
            await userEvent.click(
                screen.getByRole('button', {
                    name: manual ? '新增任務' : '儲存訂閱',
                })
            );
            await waitFor(() => {
                expect(onSaved).toHaveBeenCalledOnce();
            });
            expect(fetch).toHaveBeenCalledTimes(2);
            expect(onClose).toHaveBeenCalledOnce();
        }
    );
    it.each([true, false])(
        'creates a folder without submitting the parent draft (manual=%s)',
        async (manual) => {
            const fetch = vi
                .spyOn(globalThis, 'fetch')
                .mockImplementation(async (input, options) => {
                    const path = requestPath(input);
                    if (
                        path === '/api/pikpak/folders' &&
                        options?.method === 'POST'
                    ) {
                        return Response.json({
                            folder: { id: 'fixture-folder' },
                        });
                    }
                    if (path.startsWith('/api/pikpak/folders?')) {
                        return Response.json(
                            listing(
                                new URL(
                                    path,
                                    'https://fixture.test'
                                ).searchParams.get('parent_id') || ''
                            )
                        );
                    }
                    return Response.json({ ok: true });
                });
            const saved = vi.fn(success);
            const { onClose } = renderDraft(manual, saved);
            const child = await openFolderDialog();
            expect(child.closest('form')).toBeNull();
            expect(document.querySelector('form form')).toBeNull();
            fireEvent.change(within(child).getByLabelText('資料夾名稱'), {
                target: { value: 'Fixture folder' },
            });
            await userEvent.click(
                within(child).getByRole('button', { name: '建立資料夾' })
            );
            await waitFor(() => {
                expect(
                    screen.queryByRole('dialog', { name: '新增資料夾' })
                ).toBeNull();
            });
            const writes = () =>
                fetch.mock.calls.filter(
                    ([, options]) => options?.method === 'POST'
                );
            expect(writes().map(([input]) => requestPath(input))).toEqual([
                '/api/pikpak/folders',
            ]);
            expect(saved).not.toHaveBeenCalled();
            expect(onClose).not.toHaveBeenCalled();
            await userEvent.click(
                await screen.findByRole('button', { name: '使用此資料夾' })
            );
            await userEvent.click(
                screen.getByRole('button', {
                    name: manual ? '新增任務' : '儲存訂閱',
                })
            );
            await waitFor(() => {
                expect(saved).toHaveBeenCalledOnce();
            });
            expect(writes().map(([input]) => requestPath(input))).toEqual([
                '/api/pikpak/folders',
                manual ? '/api/jobs' : '/api/subscriptions',
            ]);
            expect(JSON.parse(writes()[1][1]?.body as string)).toMatchObject({
                destination: 'Fixture folder',
                destination_id: 'fixture-folder',
                destination_account_ref: 'fixture-account',
            });
        }
    );
});
