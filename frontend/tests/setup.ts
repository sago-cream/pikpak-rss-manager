import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

Object.defineProperty(globalThis, 'matchMedia', {
    value: vi.fn(() => ({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
    })),
});
HTMLDialogElement.prototype.showModal = function showModal() {
    this.setAttribute('open', '');
};
HTMLElement.prototype.hidePopover = function hidePopover() {
    delete this.dataset.open;
};
HTMLElement.prototype.hasPointerCapture = () => false;
HTMLElement.prototype.scrollIntoView = () => undefined;
afterEach(() => {
    cleanup();
    localStorage.clear();
});
