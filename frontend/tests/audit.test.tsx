import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { Dialog } from '../src/components/Dialog.js';
import { JobDialog } from '../src/components/JobDialog.js';
import { Preferences } from '../src/components/Preferences.js';
import { Settings } from '../src/components/Settings.js';
import { SubscriptionDialog } from '../src/components/SubscriptionDialog.js';
import { Tasks } from '../src/components/Tasks.js';
import { Workspace } from '../src/components/Workspace.js';
import type { Job, Subscription } from '../src/lib/types.js';

const success = async () => undefined;
const response = (value: unknown) => Response.json(value);
const job = (id: string, state: string): Job => ({
    id,
    state,
    subscription_id: 1,
    title: 'episode.mkv',
    rule: { title: 'Series', regex: '', replacement: '', rename_enabled: true },
    destination: 'Series',
    task_id: `cloud-${id}`,
    progress: 0,
    created_at: 1,
    error: '',
});
const sub = (id: number): Subscription => ({
    id,
    name: `Series ${id}`,
    rss_url: 'https://fixture.test/rss',
    destination: '',
    interval_minutes: 10,
    enabled: true,
    initialized: true,
    last_checked: 0,
    next_check: 0,
    last_error: '',
    regex: '',
    replacement: '',
    rename_enabled: false,
});

describe('Workspace audit interactions', () => {
    it('shows every task by default with colored Lucide icons and a cleanup action', () => {
        render(
            <Preferences>
                <Tasks
                    busy={false}
                    date={() => '12:00'}
                    jobs={Array.from({ length: 13 }, (_, id) =>
                        job(String(id), id === 0 ? 'complete' : 'queued')
                    )}
                    onAdd={vi.fn<() => void>()}
                    onClear={vi.fn<() => void>()}
                    onOpen={vi.fn<() => void>()}
                />
            </Preferences>
        );
        expect(document.querySelectorAll('[data-job]')).toHaveLength(13);
        expect(screen.queryByText('顯示更多')).toBeNull();
        expect(
            document.querySelector('.job-indicator.green .lucide-circle-check')
        ).toBeTruthy();
        expect(
            document.querySelector('.job-indicator.purple .lucide-download')
        ).toBeTruthy();
        expect(
            document.querySelector(
                '#clear-completed-jobs .lucide-brush-cleaning'
            )
        ).toBeTruthy();
        expect(document.querySelector('.job-indicator .dot')).toBeNull();
    });
    it('guards changed subscription drafts on Cancel and Escape, and disables Save again when restored', async () => {
        const confirm = vi.spyOn(globalThis, 'confirm').mockReturnValue(false);
        const close = vi.fn<() => void>();
        render(
            <Preferences>
                <SubscriptionDialog
                    onClose={close}
                    onSaved={success}
                    status={{ connected: true }}
                    subscription={sub(1)}
                />
            </Preferences>
        );
        expect(screen.getByRole('button', { name: '儲存訂閱' })).toHaveProperty(
            'disabled',
            true
        );
        fireEvent.change(screen.getByLabelText('訂閱名稱'), {
            target: { value: 'Draft' },
        });
        expect(screen.getByRole('button', { name: '儲存訂閱' })).toHaveProperty(
            'disabled',
            false
        );
        await userEvent.click(screen.getByRole('button', { name: '取消' }));
        fireEvent(
            document.querySelector('dialog')!,
            new Event('cancel', { cancelable: true })
        );
        expect(confirm).toHaveBeenCalledTimes(2);
        expect(close).not.toHaveBeenCalled();
        await userEvent.click(screen.getByRole('button', { name: '檔名規則' }));
        await userEvent.click(screen.getByRole('button', { name: '訂閱設定' }));
        expect(screen.getByLabelText('訂閱名稱')).toHaveProperty(
            'value',
            'Draft'
        );
        fireEvent.change(screen.getByLabelText('訂閱名稱'), {
            target: { value: 'Series 1' },
        });
        expect(screen.getByRole('button', { name: '儲存訂閱' })).toHaveProperty(
            'disabled',
            true
        );
        await userEvent.click(screen.getByRole('button', { name: '關閉' }));
        expect(confirm).toHaveBeenCalledTimes(2);
        expect(close).toHaveBeenCalledOnce();
    });
    it('protects site drafts after switching categories and clears the dirty baseline after saving', async () => {
        vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) =>
            response(
                (typeof input === 'string'
                    ? input
                    : input instanceof URL
                      ? input.href
                      : input.url) === '/api/session'
                    ? { csrf: 'fixture' }
                    : {
                          public_url: 'https://fixture.test',
                          allow_private_feeds: false,
                      }
            )
        );
        const confirm = vi.spyOn(globalThis, 'confirm').mockReturnValue(false);
        const close = vi.fn<() => void>();
        render(
            <Preferences>
                <Dialog id='test-settings' onClose={close} title='Settings'>
                    <Settings
                        onAuthenticated={success}
                        onRefresh={success}
                        status={{ connected: true }}
                    />
                </Dialog>
            </Preferences>
        );
        await userEvent.click(screen.getByRole('button', { name: '網站設定' }));
        await screen.findByLabelText('網站網址');
        expect(
            screen.getByRole('button', { name: '儲存網站設定' })
        ).toHaveProperty('disabled', true);
        fireEvent.change(screen.getByLabelText('網站網址'), {
            target: { value: 'https://changed.test' },
        });
        await userEvent.click(
            document.querySelector<HTMLButtonElement>('#settings-nav-password')!
        );
        await userEvent.click(screen.getByRole('button', { name: '關閉' }));
        expect(close).not.toHaveBeenCalled();
        expect(confirm).toHaveBeenCalledOnce();
        await userEvent.click(screen.getByRole('button', { name: '網站設定' }));
        await userEvent.click(
            screen.getByRole('button', { name: '儲存網站設定' })
        );
        await screen.findByText('網站設定已儲存');
        expect(
            screen.getByRole('button', { name: '儲存網站設定' })
        ).toHaveProperty('disabled', true);
        await userEvent.click(screen.getByRole('button', { name: '關閉' }));
        expect(confirm).toHaveBeenCalledOnce();
        expect(close).toHaveBeenCalledOnce();
    });
    it('leads task details with the result, prioritizes attention files and reconciles uncertain submissions', async () => {
        const fetch = vi
            .spyOn(globalThis, 'fetch')
            .mockImplementation(async () =>
                response({
                    job: job('uncertain', 'submission_unknown'),
                    files: [
                        {
                            file_id: 'ok',
                            state: 'done',
                            original_name: 'done.mkv',
                            target_name: 'done.mkv',
                        },
                        {
                            file_id: 'review',
                            state: 'review',
                            original_name: 'review.mkv',
                            target_name: 'rename.mkv',
                            error: 'Conflict',
                        },
                    ],
                })
            );
        const close = vi.fn<() => void>();
        render(
            <Preferences>
                <JobDialog
                    jobID='uncertain'
                    onChanged={success}
                    onClose={close}
                />
            </Preferences>
        );
        await screen.findByRole('dialog', { name: 'Series' });
        const files = [...document.querySelectorAll('.file-detail strong')].map(
            (e) => e.textContent
        );
        expect(files).toEqual(['review.mkv', 'done.mkv']);
        expect(document.querySelector('.technical-details')).toHaveProperty(
            'open',
            false
        );
        expect(document.querySelector('.task-completed-files')).toHaveProperty(
            'open',
            false
        );
        await userEvent.click(
            screen.getByRole('button', { name: '重新核對結果' })
        );
        await waitFor(() => {
            expect(close).toHaveBeenCalledOnce();
        });
        expect(fetch.mock.calls[1][0]).toBe('/api/jobs/uncertain/retry');
        expect(fetch.mock.calls[1][1]).toHaveProperty('method', 'POST');
    });
    it.each([
        ['needs_review', '重試改名'],
        ['paused_auth', '修復連線'],
        ['failed', '刪除任務'],
    ])('offers a specific action for %s', async (state, label) => {
        vi.spyOn(globalThis, 'fetch').mockResolvedValue(
            response({ job: job('task', state), files: [] })
        );
        const fix = vi.fn<() => void>();
        render(
            <Preferences>
                <JobDialog
                    jobID='task'
                    onChanged={success}
                    onClose={vi.fn<() => void>()}
                    onFixConnection={fix}
                />
            </Preferences>
        );
        const action = await screen.findByRole('button', { name: label });
        if (state === 'paused_auth') {
            expect(screen.queryByText('接續下載')).toBeNull();
            await userEvent.click(action);
            expect(fix).toHaveBeenCalledOnce();
        }
    });
    it('keeps the task filter when switching mobile sections and restores focus to the opening row', async () => {
        globalThis.history.replaceState(null, '', '#overview');
        vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
            const path =
                typeof input === 'string'
                    ? input
                    : input instanceof URL
                      ? input.href
                      : input.url;
            return response(
                path === '/api/jobs'
                    ? [job('done', 'complete'), job('queued', 'queued')]
                    : path.startsWith('/api/jobs/')
                      ? { job: job('queued', 'queued'), files: [] }
                      : path === '/api/subscriptions' || path === '/api/events'
                        ? []
                        : { connected: true }
            );
        });
        render(
            <Preferences>
                <Workspace onAuthenticated={success} version='test' />
            </Preferences>
        );
        const tabs = await screen.findByRole('button', {
            name: '任務',
        });
        await userEvent.click(tabs);
        await userEvent.click(
            screen.getByRole('combobox', { name: '任務狀態' })
        );
        await userEvent.click(screen.getByRole('option', { name: '進行中' }));
        await userEvent.click(screen.getByRole('button', { name: '訂閱' }));
        await userEvent.click(tabs);
        expect(
            document.querySelector<HTMLElement>('.workspace-body')?.dataset
                .mobileView
        ).toBe('jobs');
        expect(
            screen.getByRole('combobox', { name: '任務狀態' }).textContent
        ).toBe('進行中');
        const opener = document.querySelector<HTMLButtonElement>(
            '[data-job="queued"]'
        )!;
        vi.spyOn(opener, 'getClientRects').mockReturnValue([
            new DOMRect(),
        ] as unknown as DOMRectList);
        await userEvent.click(opener);
        await screen.findByRole('dialog', { name: 'Series' });
        await userEvent.click(screen.getByRole('button', { name: '關閉' }));
        await waitFor(() => {
            expect(document.activeElement).toBe(opener);
        });
    });
    it('scopes pending tracking actions per subscription and prevents duplicate submissions', async () => {
        globalThis.history.replaceState(null, '', '#overview');
        const writes = [
            Promise.withResolvers<Response>(),
            Promise.withResolvers<Response>(),
        ];
        const subscriptions = [sub(1), sub(2)];
        const fetch = vi
            .spyOn(globalThis, 'fetch')
            .mockImplementation(async (input, options) => {
                const path =
                    typeof input === 'string'
                        ? input
                        : input instanceof URL
                          ? input.href
                          : input.url;
                if (options?.method === 'PUT') {
                    const id = Number(path.split('/').at(-1));
                    return await writes[id - 1].promise;
                }
                return response(
                    path === '/api/subscriptions'
                        ? subscriptions
                        : path === '/api/jobs' || path === '/api/events'
                          ? []
                          : { connected: true }
                );
            });
        render(
            <Preferences>
                <Workspace onAuthenticated={success} version='test' />
            </Preferences>
        );
        const first = await screen.findByRole('button', {
            name: '暫停追蹤 Series 1',
        });
        const second = screen.getByRole('button', {
            name: '暫停追蹤 Series 2',
        });
        await userEvent.click(first);
        expect(first).toHaveProperty('disabled', true);
        expect(first.getAttribute('aria-busy')).toBe('true');
        expect(second).toHaveProperty('disabled', false);
        await userEvent.click(first);
        await userEvent.click(second);
        expect(
            fetch.mock.calls.filter(([, options]) => options?.method === 'PUT')
        ).toHaveLength(2);
        subscriptions[0].enabled = false;
        writes[0].resolve(response({ ok: true }));
        await screen.findByRole('button', { name: '繼續追蹤 Series 1' });
        expect(second).toHaveProperty('disabled', true);
        subscriptions[1].enabled = false;
        writes[1].resolve(response({ ok: true }));
        await waitFor(() => {
            expect(
                screen.getByRole('button', { name: '繼續追蹤 Series 2' })
            ).toHaveProperty('disabled', false);
        });
    });
});
