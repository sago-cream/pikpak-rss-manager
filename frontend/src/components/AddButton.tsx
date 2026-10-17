import type { JSX } from 'react';

import { Icon } from './Icon.js';
import { usePreferences } from './Preferences.js';

export function AddButton({
    kind,
    onClick,
}: {
    kind: 'task' | 'subscription';
    onClick: () => void;
}): JSX.Element {
    const { t } = usePreferences();
    const label = t(kind === 'task' ? '新增任務' : '新增訂閱');
    return (
        <button
            aria-label={label}
            className={`icon-button section-add${kind === 'subscription' ? ' new-subscription' : ''}`}
            data-manual-task={kind === 'task' ? true : undefined}
            onClick={onClick}
            title={label}
            type='button'
        >
            <Icon name='plus' />
        </button>
    );
}
