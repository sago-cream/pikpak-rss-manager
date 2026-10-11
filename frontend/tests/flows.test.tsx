import { StrictMode } from 'react';
import {
    act,
    fireEvent,
    render,
    screen,
    waitFor,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { App } from '../src/components/App.js';
import { Auth } from '../src/components/Auth.js';
import { BackfillDialog } from '../src/components/BackfillDialog.js';
import { FolderPicker } from '../src/components/FolderPicker.js';
import { Preferences } from '../src/components/Preferences.js';
import { RenamePreview } from '../src/components/RenamePreview.js';
import { Settings } from '../src/components/Settings.js';
import { SubscriptionDialog } from '../src/components/SubscriptionDialog.js';
import { Tasks } from '../src/components/Tasks.js';
import { TokenForm } from '../src/components/TokenForm.js';
import { Workspace } from '../src/components/Workspace.js';
import { api, setCSRF } from '../src/lib/api.js';
import { translate } from '../src/lib/i18n.js';
import { currentFilename } from '../src/lib/jobs.js';
import type { Job, Rule } from '../src/lib/types.js';

function response(value: unknown, status = 200): Response {
    return Response.json(value, {
        status,
        headers: { 'Content-Type': 'application/json' },
    });
}
const success = async () => undefined;
const job = (id: string, state: string): Job => ({
    id,
    state,
    subscription_id: 1,
    title: '使用者.mkv',
    rule: {
        title: '使用者',
        regex: '',
        replacement: '',
        rename_enabled: false,
    },
    destination: '',
    task_id: `cloud-${id}`,
    progress: 0,
    created_at: 1,
    error: '',
});
describe('React flows', () => {
    it('opens existing settings links after loading the workspace', async () => {
        globalThis.history.replaceState(null, '', '#settings');
        vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
            const path =
                typeof input === 'string'
                    ? input
                    : input instanceof URL
                      ? input.href
                      : input.url;
            return response(
                path === '/api/settings/pikpak'
                    ? { connected: true }
                    : path === '/api/settings/app'
                      ? {
                            public_url: 'http://localhost:8080',
                            allow_private_rss: false,
                        }
                      : []
            );
        });
        try {
            render(
                <Preferences>
                    <Workspace onAuthenticated={success} version='fixture' />
                </Preferences>
            );
            expect(
                await screen.findByRole('dialog', { name: '系統設定' })
            ).toBeTruthy();
            expect(
                screen.getByRole('navigation', { name: '設定分類' })
            ).toBeTruthy();
        } finally {
            globalThis.history.replaceState(null, '', '/');
        }
    });
    it('does not close a newer dialog when a dismissed form finishes saving', async () => {
        const pending = Promise.withResolvers<Response>();
        vi.spyOn(globalThis, 'fetch').mockReturnValue(pending.promise);
        const close = vi.fn<() => void>();
        const saved = vi.fn(success);
        const view = render(
            <Preferences>
                <SubscriptionDialog
                    manual
                    onClose={close}
                    onSaved={saved}
                    status={{ connected: true }}
                />
            </Preferences>
        );
        fireEvent.change(screen.getByLabelText('下載連結'), {
            target: { value: 'https://public.test/file' },
        });
        fireEvent.submit(document.querySelector('#subscription-form')!);
        view.unmount();
        pending.resolve(response({ ok: true }));
        await waitFor(() => {
            expect(saved).toHaveBeenCalledOnce();
        });
        expect(close).not.toHaveBeenCalled();
    });
    it('honors the server recovery bootstrap without exposing account data', async () => {
        const root = document.createElement('div');
        root.id = 'root';
        root.dataset.page = 'recovery';
        document.body.append(root);
        const fetch = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
            response({
                initialized: true,
                authenticated: true,
                csrf: 'fixture-csrf',
                version: 'fixture',
            })
        );
        const view = render(
            <Preferences>
                <App />
            </Preferences>,
            { container: root }
        );
        expect(
            await screen.findByRole('heading', { name: '忘記密碼' })
        ).toBeTruthy();
        expect(fetch).toHaveBeenCalledTimes(1);
        view.unmount();
        root.remove();
    });
    it('aborts pending folder and source requests when their components close', async () => {
        const signals: AbortSignal[] = [];
        vi.spyOn(globalThis, 'fetch').mockImplementation(
            async (_input, options) =>
                await new Promise<Response>((_resolve, reject) => {
                    const signal = options?.signal;
                    if (signal) {
                        signals.push(signal);
                        signal.addEventListener('abort', () => {
                            reject(new DOMException('Cancelled', 'AbortError'));
                        });
                    }
                })
        );
        const folders = render(
            <Preferences>
                <FolderPicker
                    initialID=''
                    onClose={vi.fn<() => void>()}
                    onSelect={vi.fn<() => void>()}
                />
            </Preferences>
        );
        await waitFor(() => {
            expect(signals.length).toBe(1);
        });
        folders.unmount();
        expect(signals[0].aborted).toBe(true);
        const preview = render(
            <Preferences>
                <RenamePreview
                    rule={{
                        title: '',
                        regex: '',
                        replacement: '',
                        rename_enabled: true,
                    }}
                    subscriptionID={0}
                    url='https://public.test/rss'
                />
            </Preferences>
        );
        await waitFor(() => {
            expect(signals).toHaveLength(2);
        });
        preview.unmount();
        expect(signals[1].aborted).toBe(true);
    });
    it('debounces enabled source reads, skips invalid URLs and ignores cancelled responses', async () => {
        vi.useFakeTimers();
        const oldSource = Promise.withResolvers<Response>();
        const signals: AbortSignal[] = [];
        const fetch = vi
            .spyOn(globalThis, 'fetch')
            .mockImplementation(async (_input, options) => {
                signals.push(options!.signal!);
                return signals.length === 1
                    ? await oldSource.promise
                    : response({
                          items: [
                              {
                                  kind: 'torrent_file',
                                  title: 'new',
                                  filename: 'new.mkv',
                              },
                          ],
                          notices: [],
                      });
            });
        const view = render(
            <Preferences>
                <SubscriptionDialog
                    onClose={vi.fn<() => void>()}
                    onSaved={success}
                    status={{ connected: true }}
                />
            </Preferences>
        );
        try {
            fireEvent.change(screen.getByLabelText('RSS 連結'), {
                target: { value: 'https://public.test/old' },
            });
            await act(async () => {
                await vi.advanceTimersByTimeAsync(600);
            });
            expect(fetch).not.toHaveBeenCalled();
            fireEvent.click(screen.getByLabelText('重命名檔案'));
            expect(
                screen.getByText(
                    '支援 $1、${name}；$$ 表示字面 $。請保留需要的副檔名。'
                )
            ).toBeTruthy();
            await act(async () => {
                await vi.advanceTimersByTimeAsync(499);
            });
            expect(fetch).not.toHaveBeenCalled();
            await act(async () => {
                await vi.advanceTimersByTimeAsync(1);
            });
            expect(fetch).toHaveBeenCalledOnce();
            fireEvent.change(screen.getByLabelText('RSS 連結'), {
                target: { value: 'not-a-url' },
            });
            expect(signals[0].aborted).toBe(true);
            expect(
                screen.getByRole('button', { name: '重新整理種子檔名' })
            ).toHaveProperty('disabled', true);
            await act(async () => {
                await vi.advanceTimersByTimeAsync(600);
            });
            expect(fetch).toHaveBeenCalledOnce();
            fireEvent.change(screen.getByLabelText('RSS 連結'), {
                target: { value: 'https://public.test/new' },
            });
            await act(async () => {
                await vi.advanceTimersByTimeAsync(500);
            });
            expect(fetch).toHaveBeenCalledTimes(2);
            await act(async () => {
                oldSource.resolve(
                    response({
                        items: [
                            {
                                kind: 'torrent_file',
                                title: 'old',
                                filename: 'old.mkv',
                            },
                        ],
                        notices: [],
                    })
                );
            });
            expect(screen.getByRole('option').textContent).toBe('new.mkv');
            expect(screen.queryByText('old.mkv')).toBeNull();
        } finally {
            view.unmount();
            vi.useRealTimers();
        }
    });
    it('rejects setup mismatch without an API mutation and clears passwords', async () => {
        const fetch = vi.spyOn(globalThis, 'fetch');
        render(
            <Preferences>
                <Auth onAuthenticated={success} page='setup' version='test' />
            </Preferences>
        );
        fireEvent.change(screen.getByLabelText('密碼'), {
            target: { value: 'one' },
        });
        fireEvent.change(screen.getByLabelText('再次輸入密碼'), {
            target: { value: 'two' },
        });
        fireEvent.submit(document.querySelector('#setup-form')!);
        expect(await screen.findByRole('alert')).toHaveProperty(
            'textContent',
            '兩次輸入的密碼不一致'
        );
        expect(fetch).not.toHaveBeenCalled();
        expect(screen.getByLabelText('密碼')).toHaveProperty('value', '');
        const label = document.querySelector('label[for="setup-public-url"]')!;
        expect(label.nextElementSibling?.className).toBe('info-trigger');
        expect(label.parentElement?.nextElementSibling?.tagName).toBe('INPUT');
    });
    it('shares PAT entry, clears failed secrets and makes only Create PAT a link', async () => {
        vi.spyOn(globalThis, 'fetch').mockResolvedValue(
            response({ error: '請輸入 PikPak PAT' }, 400)
        );
        render(
            <Preferences>
                <TokenForm onboarding onSaved={success} />
            </Preferences>
        );
        fireEvent.change(screen.getByLabelText('個人存取權杖（PAT）'), {
            target: { value: 'fixture-secret' },
        });
        fireEvent.submit(document.querySelector('#token-form')!);
        expect(await screen.findByRole('alert')).toHaveProperty(
            'textContent',
            '請輸入 PikPak PAT'
        );
        expect(screen.getByLabelText('個人存取權杖（PAT）')).toHaveProperty(
            'value',
            ''
        );
        expect(screen.getByRole('link').textContent).toBe('建立 PAT');
    });
    it('recovers with the existing server API and clears all secrets', async () => {
        const fetch = vi
            .spyOn(globalThis, 'fetch')
            .mockResolvedValue(response({ ok: true }));
        render(
            <Preferences>
                <Auth
                    onAuthenticated={success}
                    page='recovery'
                    version='test'
                />
            </Preferences>
        );
        for (const [label, value] of [
            ['復原碼', ' fixture-code '],
            ['新密碼', 'x'],
            ['再次輸入新密碼', 'x'],
        ]) {
            fireEvent.change(screen.getByLabelText(label), {
                target: { value },
            });
        }
        fireEvent.submit(document.querySelector('#recovery-form')!);
        expect(await screen.findByText('密碼已重設')).toBeTruthy();
        expect(JSON.parse(fetch.mock.calls[0][1]?.body as string)).toEqual({
            code: 'fixture-code',
            new_password: 'x',
            confirm_password: 'x',
        });
        expect(screen.queryByLabelText('復原碼')).toBeNull();
    });
    it('retains unsaved site fields when switching settings categories', async () => {
        vi.spyOn(globalThis, 'fetch').mockResolvedValue(
            response({
                public_url: 'http://localhost:8080',
                allow_private_feeds: false,
            })
        );
        render(
            <Preferences>
                <Settings
                    onAuthenticated={success}
                    onRefresh={success}
                    status={{ connected: true }}
                />
            </Preferences>
        );
        await waitFor(() => {
            expect(document.querySelector('#settings-public-url')).toBeTruthy();
        });
        await userEvent.click(screen.getByRole('button', { name: '網站設定' }));
        fireEvent.change(screen.getByLabelText('網站網址'), {
            target: { value: 'https://edited.test' },
        });
        await userEvent.click(
            document.querySelector<HTMLButtonElement>('#settings-nav-password')!
        );
        await userEvent.click(screen.getByRole('button', { name: '網站設定' }));
        expect(screen.getByLabelText('網站網址')).toHaveProperty(
            'value',
            'https://edited.test'
        );
    });
    it('prioritizes review tasks, opens details with pointer and keyboard, and retains filter focus', async () => {
        const onOpen = vi.fn<(id: string) => void>();
        render(
            <Preferences>
                <Tasks
                    busy={false}
                    date={() => ''}
                    jobs={[
                        job('done', 'complete'),
                        job('running', 'queued'),
                        job('review', 'needs_review'),
                    ]}
                    onAdd={vi.fn<() => void>()}
                    onClear={vi.fn<() => void>()}
                    onOpen={onOpen}
                />
            </Preferences>
        );
        expect(
            [...document.querySelectorAll<HTMLElement>('[data-job]')].map(
                (element) => element.dataset.job
            )
        ).toEqual(['review', 'running', 'done']);
        const firstTask = screen.getAllByRole('button', {
            name: '使用者 · 使用者.mkv',
        })[0];
        await userEvent.click(firstTask);
        firstTask.focus();
        await userEvent.keyboard('{Enter}');
        await userEvent.keyboard(' ');
        expect(onOpen.mock.calls).toEqual([['review'], ['review'], ['review']]);
        expect(screen.queryByText('查看')).toBeNull();
        const select = screen.getByLabelText('任務狀態');
        await userEvent.selectOptions(select, 'complete');
        expect(
            screen.getByRole('button', { name: '清除已完成任務' })
        ).toBeTruthy();
        await userEvent.selectOptions(select, 'complete');
        expect(
            document.querySelector<HTMLElement>('[data-job]')?.dataset.job
        ).toBe('done');
        select.focus();
        fireEvent.change(select, { target: { value: 'running' } });
        expect(document.activeElement).toBe(select);
    });
    it('submits manual downloads to root without subscription or rename fields', async () => {
        const fetch = vi
            .spyOn(globalThis, 'fetch')
            .mockResolvedValue(response({ ok: true }));
        const saved = vi.fn(success);
        render(
            <Preferences>
                <SubscriptionDialog
                    manual
                    onClose={vi.fn<() => void>()}
                    onSaved={saved}
                    status={{ connected: true }}
                />
            </Preferences>
        );
        fireEvent.change(screen.getByLabelText('下載連結'), {
            target: { value: 'magnet:?xt=urn:btih:fixture' },
        });
        fireEvent.submit(document.querySelector('#subscription-form')!);
        await waitFor(() => {
            expect(saved).toHaveBeenCalled();
        });
        expect(fetch.mock.calls[0][0]).toBe('/api/jobs');
        expect(JSON.parse(fetch.mock.calls[0][1]?.body as string)).toEqual({
            url: 'magnet:?xt=urn:btih:fixture',
            destination: '',
            destination_id: '',
            destination_account_ref: '',
        });
        expect(screen.queryByLabelText('重命名檔案')).toBeNull();
    });
    it('selects root with its account reference and aborts stale folder reads', async () => {
        vi.spyOn(globalThis, 'fetch').mockResolvedValue(
            response({
                current: { id: '', name: '根目錄', path: '' },
                breadcrumbs: [],
                folders: [],
                account_ref: 'opaque-account',
            })
        );
        const selected = vi.fn<() => void>();
        const view = render(
            <Preferences>
                <FolderPicker
                    initialID=''
                    onClose={vi.fn<() => void>()}
                    onSelect={selected}
                />
            </Preferences>
        );
        await userEvent.click(
            await screen.findByRole('button', { name: '使用根目錄' })
        );
        expect(selected).toHaveBeenCalledWith({
            path: '',
            id: '',
            account: 'opaque-account',
        });
        view.unmount();
    });
    it('keeps backfill selection and confirmation token for an explicit idempotent retry', async () => {
        const fetch = vi
            .spyOn(globalThis, 'fetch')
            .mockImplementation(async (input) =>
                (typeof input === 'string'
                    ? input
                    : input instanceof URL
                      ? input.href
                      : input.url
                ).endsWith('/preview')
                    ? response({
                          token: 'confirmation-token',
                          items: [
                              {
                                  id: 'torrent-1',
                                  title: '公開測試',
                                  filenames: ['raw.mkv'],
                              },
                          ],
                          notices: [],
                      })
                    : response({ error: '操作未完成，請稍後再試' }, 400)
            );
        vi.spyOn(globalThis, 'confirm').mockReturnValue(true);
        render(
            <Preferences>
                <BackfillDialog
                    onClose={vi.fn<() => void>()}
                    onSaved={success}
                    subscriptionID={1}
                />
            </Preferences>
        );
        await userEvent.click(await screen.findByRole('checkbox'));
        await userEvent.click(
            screen.getByRole('button', { name: '下載所選項目' })
        );
        await waitFor(() => {
            expect(
                screen.getByRole('button', { name: '下載所選項目' })
            ).toHaveProperty('disabled', false);
        });
        await userEvent.click(
            screen.getByRole('button', { name: '下載所選項目' })
        );
        await waitFor(() => {
            expect(fetch.mock.calls).toHaveLength(3);
        });
        const payloads = fetch.mock.calls
            .slice(1)
            .map((call) => JSON.parse(call[1]?.body as string) as unknown);
        expect(payloads).toEqual([
            { token: 'confirmation-token', selected: ['torrent-1'] },
            { token: 'confirmation-token', selected: ['torrent-1'] },
        ]);
    });
    it('automatically loads filenames once in Strict Mode and preserves selection through refresh and retry', async () => {
        let reads = 0;
        const items = [
            { kind: 'title', title: 'ignore', filename: 'guessed.mkv' },
            { kind: 'torrent_file', title: 'raw', filename: 'raw[01].mkv' },
            { kind: 'torrent_file', title: 'raw', filename: 'raw[01].mkv' },
            { kind: 'torrent_file', title: 'raw', filename: 'raw[02].mkv' },
        ];
        const fetch = vi
            .spyOn(globalThis, 'fetch')
            .mockImplementation(async (input, options) => {
                const path =
                    typeof input === 'string'
                        ? input
                        : input instanceof URL
                          ? input.href
                          : input.url;
                if (path.endsWith('/samples')) {
                    reads++;
                    return reads === 2
                        ? response({ error: '來源回傳非成功狀態' }, 400)
                        : response({
                              items: reads > 2 ? items.toReversed() : items,
                              notices: [],
                          });
                }
                const body = JSON.parse(options?.body as string) as {
                    filename: string;
                    rule: Rule;
                };
                return response({
                    name: body.filename.replace(
                        new RegExp(body.rule.regex),
                        body.rule.replacement
                    ),
                });
            });
        const rule: Rule = {
            title: '',
            regex: String.raw`\[(\d+)\]`,
            replacement: 'S01E$1',
            rename_enabled: true,
        };
        const view = render(
            <StrictMode>
                <Preferences>
                    <RenamePreview
                        rule={rule}
                        subscriptionID={1}
                        url='https://public.test/rss'
                    />
                </Preferences>
            </StrictMode>
        );
        const options = await screen.findAllByRole('option');
        expect(options.map((option) => option.textContent)).toEqual([
            'raw[01].mkv',
            'raw[02].mkv',
        ]);
        expect(await screen.findByText('rawS01E01.mkv')).toBeTruthy();
        expect(reads).toBe(1);
        await userEvent.selectOptions(screen.getByLabelText('種子檔名'), '1');
        expect(await screen.findByText('rawS01E02.mkv')).toBeTruthy();
        await userEvent.click(
            screen.getByRole('button', { name: '重新整理種子檔名' })
        );
        expect(await screen.findByText('來源回傳非成功狀態')).toBeTruthy();
        expect(screen.getByLabelText('種子檔名')).toHaveProperty('value', '1');
        await userEvent.click(
            screen.getByRole('button', { name: '重新整理種子檔名' })
        );
        await waitFor(() => {
            expect(screen.getByLabelText('種子檔名')).toHaveProperty(
                'value',
                '0'
            );
        });
        expect(screen.getByText('rawS01E02.mkv')).toBeTruthy();
        view.rerender(
            <StrictMode>
                <Preferences>
                    <RenamePreview
                        rule={{ ...rule, replacement: 'S02E$1' }}
                        subscriptionID={1}
                        url='https://public.test/rss'
                    />
                </Preferences>
            </StrictMode>
        );
        expect(await screen.findByText('rawS02E02.mkv')).toBeTruthy();
        expect(reads).toBe(3);
        expect(
            fetch.mock.calls.filter((call) => call[0] === '/api/rules/preview')
        ).toHaveLength(3);
        view.unmount();
    });
    it('preserves actual filenames and translates application copy with literal regex references', () => {
        expect(
            currentFilename({
                actual_name: 'PikPak (1).mkv',
                original_name: 'raw.mkv',
                target_name: 'requested.mkv',
            })
        ).toBe('PikPak (1).mkv');
        expect(translate('每 10 分鐘', 'en')).toBe('Every 10 min');
        expect(translate('使用者.mkv', 'en')).toBe('使用者.mkv');
        expect(
            translate(
                '支援 $1、${name}；$$ 表示字面 $。請保留需要的副檔名。',
                'en'
            )
        ).toContain('${name}');
    });
    it('sends session CSRF and expires authenticated UI on an unauthorized response', async () => {
        setCSRF('fixture-csrf');
        const expired = vi.fn<() => void>();
        globalThis.addEventListener('session-expired', expired);
        const fetch = vi
            .spyOn(globalThis, 'fetch')
            .mockResolvedValue(response({ error: '請先登入' }, 401));
        await expect(api('/api/subscriptions')).rejects.toThrow('請先登入');
        expect(fetch.mock.calls[0][1]?.headers).toHaveProperty(
            'X-CSRF-Token',
            'fixture-csrf'
        );
        expect(expired).toHaveBeenCalledOnce();
        globalThis.removeEventListener('session-expired', expired);
    });
});
