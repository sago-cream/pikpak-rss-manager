import { useEffect, useRef, useState } from 'react';
import type { JSX } from 'react';

import { api } from '../lib/api.js';
import { errorMessage, isCancelled } from '../lib/i18n.js';
import type {
    RenameResult,
    Rule,
    SampleResult,
    SourceSample,
} from '../lib/types.js';
import { FormField } from './FormField.js';
import { usePreferences } from './Preferences.js';

export function RenamePreview({
    url,
    subscriptionID,
    rule,
    paused = false,
}: {
    url: string;
    subscriptionID: number;
    rule: Rule;
    paused?: boolean;
}): JSX.Element {
    const { t } = usePreferences();
    const [error, setError] = useState('');
    const [notice, setNotice] = useState('');
    const [notices, setNotices] = useState<string[]>([]);
    const [renderedRule, setRenderedRule] = useState('');
    const [pending, setPending] = useState(false);
    const [result, setResult] = useState<RenameResult | null>(null);
    const [samples, setSamples] = useState<SourceSample[]>([]);
    const [selected, setSelected] = useState('0');
    const request = useRef<AbortController | null>(null);
    const filename = samples.at(Number(selected))?.filename;
    const { title, regex, rename_enabled: renameEnabled, replacement } = rule;
    const signature = JSON.stringify([
        filename,
        title,
        regex,
        replacement,
        renameEnabled,
        paused,
    ]);
    const currentResult = renderedRule === signature ? result : null;
    const currentError = renderedRule === signature ? error : '';
    useEffect(() => () => request.current?.abort(), []);
    useEffect(() => {
        if (!filename || paused) {
            return;
        }
        const controller = new AbortController();
        const timer = globalThis.setTimeout(() => {
            void api<RenameResult>(
                '/api/rules/preview',
                'POST',
                {
                    rule: {
                        title,
                        regex,
                        replacement,
                        rename_enabled: renameEnabled,
                    },
                    filename,
                },
                controller.signal
            )
                .then((value) => {
                    if (!controller.signal.aborted) {
                        setResult(value);
                        setRenderedRule(signature);
                        setError('');
                    }
                })
                .catch((error_: unknown) => {
                    if (!controller.signal.aborted && !isCancelled(error_)) {
                        setError(errorMessage(error_));
                        setRenderedRule(signature);
                        setResult(null);
                    }
                });
        }, 200);
        return () => {
            clearTimeout(timer);
            controller.abort();
        };
    }, [filename, title, regex, replacement, renameEnabled, signature, paused]);
    const load = async () => {
        if (!url.trim()) {
            setNotice('請先填寫 RSS 連結');
            return;
        }
        request.current?.abort();
        const controller = new AbortController();
        request.current = controller;
        setPending(true);
        setNotices([]);
        setNotice('正在讀取全部種子檔名…');
        try {
            const response = await api<SampleResult>(
                '/api/feeds/samples',
                'POST',
                { url: url.trim(), subscription_id: subscriptionID },
                controller.signal
            );
            if (controller.signal.aborted) {
                return;
            }
            const known = new Set<string>();
            const items = response.items.filter((sample) => {
                const key = JSON.stringify([sample.title, sample.filename]);
                if (sample.kind !== 'torrent_file' || known.has(key)) {
                    return false;
                }
                known.add(key);
                return true;
            });
            setSamples(items);
            const previous: SourceSample | undefined = samples.at(
                Number(selected)
            );
            const index = items.findIndex(
                (sample) =>
                    sample.title === previous?.title &&
                    sample.filename === previous.filename
            );
            setSelected(String(Math.max(0, index)));
            setNotices([...new Set(response.notices)]);

            setNotice(
                items.length > 0
                    ? `已讀取 ${items.length} 個種子檔名。`
                    : '目前 RSS 沒有可用的種子檔名。'
            );
        } catch (error_) {
            if (!controller.signal.aborted && !isCancelled(error_)) {
                setNotice(errorMessage(error_));
            }
        } finally {
            if (!controller.signal.aborted) {
                setPending(false);
            }
        }
    };
    return (
        <div className='preview-box'>
            <div className='preview-heading'>
                <h3>{t('檔名預覽')}</h3>
                <button
                    aria-busy={pending}
                    className='button primary'
                    disabled={pending}
                    id='source-samples'
                    onClick={() => {
                        void load();
                    }}
                    type='button'
                >
                    {t(pending ? '讀取中…' : '讀取種子檔名')}
                </button>
            </div>
            {samples.length > 0 && (
                <FormField id='preview-source' label={t('種子檔名')}>
                    <select
                        onChange={(event) => {
                            setSelected(event.target.value);
                        }}
                        value={selected}
                    >
                        {samples.map((sample, index) => (
                            <option
                                key={JSON.stringify([
                                    sample.title,
                                    sample.filename,
                                ])}
                                value={index}
                            >
                                {sample.filename}
                            </option>
                        ))}
                    </select>
                </FormField>
            )}
            <p className='field-help' id='preview-source-help' role='status'>
                {[t(notice), ...notices.map(t)].filter(Boolean).join(' ')}
            </p>
            <div
                aria-live='polite'
                className={`preview-result${currentError ? ' error' : currentResult ? ' success' : ''}`}
                id='preview-result'
            >
                {currentError ? (
                    t(currentError)
                ) : filename ? (
                    currentResult ? (
                        <>
                            <div className='rename-preview-row'>
                                <span>{t('替換結果')}</span>
                                <code>{currentResult.name}</code>
                            </div>
                            {currentResult.raw_name !== undefined &&
                                currentResult.raw_name !==
                                    currentResult.name && (
                                    <p className='preview-warning'>
                                        {currentResult.warnings
                                            ?.map(t)
                                            .join(' ')}
                                    </p>
                                )}
                        </>
                    ) : (
                        t('正在更新預覽…')
                    )
                ) : (
                    t('選擇種子檔名')
                )}
            </div>
        </div>
    );
}
