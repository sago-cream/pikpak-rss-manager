import type { JSX } from 'react';

import { jobLabels, reviewStates } from '../lib/jobs.js';
import { usePreferences } from './Preferences.js';

export function Badge({ state }: { state: string }): JSX.Element {
    const { t } = usePreferences();
    return (
        <span
            className={`badge ${state === 'complete' ? 'green' : reviewStates.includes(state) ? 'orange' : 'purple'}`}
        >
            {t(jobLabels[state] || state)}
        </span>
    );
}
