import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { App } from './components/App.js';
import { Preferences } from './components/Preferences.js';

import './styles/app.css';
import './styles/subscriptions.css';
import './styles/theme.css';

const root = document.querySelector('#root');
if (!root) {
    throw new Error('Application root missing');
}
createRoot(root).render(
    <StrictMode>
        <Preferences>
            <App />
        </Preferences>
    </StrictMode>
);
