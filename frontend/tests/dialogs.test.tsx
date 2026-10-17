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

function renderDraft(manual: boolean, onSaved = success) {
    const onClose = vi.fn<() => void>();
    const view = render(
        <Preferences>
            <SubscriptionDialog
                manual={manual}
                onClose={onClose}
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
