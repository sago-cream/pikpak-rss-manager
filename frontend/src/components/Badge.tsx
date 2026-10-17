import type { JSX } from 'react';

import { jobColor, jobLabels } from '../lib/jobs.js';
import { usePreferences } from './Preferences.js';

export function Badge({ state }: { state: string }): JSX.Element {
    const { t } = usePreferences();
    return (
        <span className={`badge ${jobColor(state)}`}>
            {t(jobLabels[state] || state)}
        </span>
    );
}
