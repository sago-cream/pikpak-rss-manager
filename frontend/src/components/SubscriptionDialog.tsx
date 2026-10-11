import { useState } from 'react';
import type { JSX } from 'react';

import folderIcon from '../assets/folder.svg';
import { api } from '../lib/api.js';
import { errorMessage } from '../lib/i18n.js';
import { useLifetime } from '../lib/lifetime.js';
import type { Rule, Status, Subscription } from '../lib/types.js';
import { Dialog } from './Dialog.js';
import { FolderPicker } from './FolderPicker.js';
import type { Destination } from './FolderPicker.js';
import { CheckboxField, Feedback, FormField } from './FormField.js';
import { usePreferences } from './Preferences.js';
import { RenamePreview } from './RenamePreview.js';

export function SubscriptionDialog({
    subscription,
    manual = false,
    status,
    onClose,
    onSaved,
}: {
    subscription?: Subscription;
    manual?: boolean;
    status: Status;
    onClose: () => void;
    onSaved: () => Promise<void>;
}): JSX.Element {
    const { t } = usePreferences();
    const active = useLifetime();
    const [browsing, setBrowsing] = useState(false);
    const [composing, setComposing] = useState(false);
    const [destination, setDestination] = useState<Destination>({
        path: subscription?.destination || '',
        id: subscription?.destination_id || '',
        account: subscription?.destination_account_ref || '',
    });
    const [enabled, setEnabled] = useState(subscription?.enabled ?? true);
    const [error, setError] = useState('');
    const [interval, setInterval] = useState(
        String(subscription?.interval_minutes ?? 10)
    );
    const [name, setName] = useState(subscription?.name || '');
    const [pending, setPending] = useState(false);
    const [regex, setRegex] = useState(
        subscription?.regex ?? String.raw`\[(\d+)\]`
    );
    const [renaming, setRenaming] = useState(
        subscription?.rename_enabled ?? false
    );
    const [replacement, setReplacement] = useState(
        subscription?.replacement ?? 'S01E$1'
    );
    const [url, setURL] = useState(subscription?.rss_url || '');
    const rule: Rule = {
        title: name.trim(),
        regex,
        replacement,
        rename_enabled: renaming,
    };
    const title = manual ? '新增任務' : subscription ? '編輯訂閱' : '新增訂閱';
    return (
        <Dialog
            id='subscription-dialog'
            onClose={onClose}
            scroll
            title={t(title)}
        >
            <form
                id='subscription-form'
                onSubmit={(event) => {
                    event.preventDefault();
                    if (pending) {
                        return;
                    }
                    setPending(true);
                    setError('');
                    const folder = {
                        destination: destination.path.trim(),
                        destination_id: destination.id,
                        destination_account_ref: destination.account,
                    };
                    const payload = manual
                        ? { url: url.trim(), ...folder }
                        : {
                              name: name.trim(),
                              rss_url: url.trim(),
                              ...folder,
                              enabled,
                              interval_minutes: Number(interval),
                              regex,
                              replacement,
                              rename_enabled: renaming,
                          };
                    const path = manual
                        ? '/api/jobs'
                        : `/api/subscriptions${subscription ? `/${subscription.id}` : ''}`;
                    void api(
                        path,
                        !manual && subscription ? 'PUT' : 'POST',
                        payload
                    )
                        .then(onSaved)
                        .then(() => {
                            if (active.current) {
                                onClose();
                            }
                        })
                        .catch((error_: unknown) => {
                            setError(errorMessage(error_));
                        })
                        .finally(() => {
                            setPending(false);
                        });
                }}
            >
                <div className='form-grid'>
                    {!manual && (
                        <FormField
                            className='full'
                            id='sub-name'
                            label={t('訂閱名稱')}
                            wrapperId='sub-name-field'
                        >
                            <input
                                maxLength={100}
                                onChange={(event) => {
                                    setName(event.target.value);
                                }}
                                onCompositionEnd={() => {
                                    setComposing(false);
                                }}
                                onCompositionStart={() => {
                                    setComposing(true);
                                }}
                                placeholder={t('訂閱名稱')}
                                required
                                value={name}
                            />
                        </FormField>
                    )}
                    <FormField
                        className='full'
                        id='sub-url'
                        label={t(manual ? '下載連結' : 'RSS 連結')}
                    >
                        <input
                            onChange={(event) => {
                                setURL(event.target.value);
                            }}
                            placeholder={
                                manual
                                    ? 'magnet:?xt=… / https://…'
                                    : 'https://…'
                            }
                            required
                            type={manual ? 'text' : 'url'}
                            value={url}
                        />
                    </FormField>
                    <div className='form-field full destination-field'>
                        <label htmlFor='sub-destination'>
                            {t('PikPak 目標資料夾')}
                        </label>
                        <div className='destination-input'>
                            <input
                                id='sub-destination'
                                onChange={(event) => {
                                    setDestination({
                                        path: event.target.value,
                                        id: '',
                                        account: '',
                                    });
                                    setBrowsing(false);
                                }}
                                placeholder={t('根目錄，或輸入 Anime/Series')}
                                value={destination.path}
                            />
                            <button
                                className='button subtle'
                                disabled={
                                    !status.connected || Boolean(status.paused)
                                }
                                id='browse-folders'
                                onClick={() => {
                                    setBrowsing((value) => !value);
                                }}
                                type='button'
                            >
                                <img
                                    alt=''
                                    className='folder-icon'
                                    src={folderIcon}
                                />
                                {t('選擇資料夾')}
                            </button>
                        </div>
                        {!status.connected && (
                            <p className='field-help'>
                                {t(
                                    '綁定 PikPak PAT 後即可列出或建立資料夾；也可先手動輸入路徑。'
                                )}
                            </p>
                        )}
                    </div>
                    {browsing && (
                        <FolderPicker
                            initialID={destination.id}
                            onClose={() => {
                                setBrowsing(false);
                            }}
                            onSelect={(value) => {
                                setDestination(value);
                                setBrowsing(false);
                            }}
                        />
                    )}
                    {!manual && (
                        <FormField
                            id='sub-interval'
                            label={t('檢查間隔（分鐘）')}
                            wrapperId='interval-field'
                        >
                            <input
                                max={10_080}
                                min={1}
                                onChange={(event) => {
                                    setInterval(event.target.value);
                                }}
                                required
                                type='number'
                                value={interval}
                            />
                        </FormField>
                    )}
                </div>
                {!manual && (
                    <>
                        <CheckboxField
                            checked={enabled}
                            id='sub-enabled'
                            label={t('啟用自動追蹤')}
                            onChange={setEnabled}
                        />
                        <CheckboxField
                            checked={renaming}
                            className='rename-toggle'
                            help={t(
                                '未勾選時保留原始檔名，只整理到目標資料夾。'
                            )}
                            id='sub-rename-enabled'
                            label={t('重命名檔案')}
                            onChange={setRenaming}
                        />
                        {renaming && (
                            <div
                                className='rule-block form-stack'
                                id='rename-options'
                            >
                                <FormField
                                    help={t(
                                        'Go RE2；不支援 lookbehind 或反向參照。'
                                    )}
                                    id='sub-regex'
                                    label={t('尋找 Regex')}
                                >
                                    <textarea
                                        className='code-input'
                                        onChange={(event) => {
                                            setRegex(event.target.value);
                                        }}
                                        onCompositionEnd={() => {
                                            setComposing(false);
                                        }}
                                        onCompositionStart={() => {
                                            setComposing(true);
                                        }}
                                        required
                                        rows={3}
                                        value={regex}
                                    />
                                </FormField>
                                <FormField
                                    help={t(
                                        '支援 $1、${name}；$$ 表示字面 $。請保留需要的副檔名。'
                                    )}
                                    id='sub-replacement'
                                    label={t('替換為')}
                                >
                                    <input
                                        className='code-input'
                                        onChange={(event) => {
                                            setReplacement(event.target.value);
                                        }}
                                        onCompositionEnd={() => {
                                            setComposing(false);
                                        }}
                                        onCompositionStart={() => {
                                            setComposing(true);
                                        }}
                                        placeholder={t('留空可移除匹配部分')}
                                        value={replacement}
                                    />
                                </FormField>
                                <RenamePreview
                                    key={url.trim()}
                                    paused={composing}
                                    rule={rule}
                                    subscriptionID={subscription?.id ?? 0}
                                    url={url}
                                />
                            </div>
                        )}
                        <p className='field-help' id='baseline-help'>
                            {t(
                                '首次檢查只建立基準。如需下載既有項目，儲存後使用「補抓」勾選並確認下載。Regex 不匹配時保留原名。'
                            )}
                        </p>
                    </>
                )}
                <Feedback message={t(error)} />
                <div className='dialog-footer'>
                    <button
                        className='button subtle'
                        onClick={onClose}
                        type='button'
                    >
                        {t('取消')}
                    </button>
                    <button
                        className='button primary'
                        disabled={pending}
                        id='save-subscription'
                        type='submit'
                    >
                        {t(manual ? '新增任務' : '儲存訂閱')}
                    </button>
                </div>
            </form>
        </Dialog>
    );
}
