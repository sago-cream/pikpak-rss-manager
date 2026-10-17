import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

import { Select } from '../src/components/Select.js';

const options = [
    { value: 'all', label: 'All' },
    { value: 'blocked', label: 'Unavailable', disabled: true },
    { value: 'active', label: 'Active' },
];

describe('Shared Select interactions', () => {
    it('selects with the keyboard, skips disabled options and returns focus on Escape', async () => {
        const change = vi.fn<(value: string) => void>();
        render(
            <Select
                aria-label='Task state'
                onValueChange={change}
                options={options}
                value='all'
            />
        );
        const control = screen.getByRole('combobox', { name: 'Task state' });
        await userEvent.tab();
        expect(document.activeElement).toBe(control);
        await userEvent.keyboard('{Enter}');
        expect(screen.getByRole('listbox')).toBeTruthy();
        await userEvent.keyboard('{ArrowDown}{Enter}');
        expect(change).toHaveBeenCalledWith('active');
        expect(screen.queryByRole('listbox')).toBeNull();
        expect(document.activeElement).toBe(control);
        await userEvent.keyboard('{Enter}{Escape}');
        expect(change).toHaveBeenCalledOnce();
        expect(document.activeElement).toBe(control);
    });
    it('does not open a disabled control', async () => {
        const change = vi.fn<(value: string) => void>();
        render(
            <Select
                aria-label='Task state'
                disabled
                onValueChange={change}
                options={options}
                value='all'
            />
        );
        await userEvent.click(
            screen.getByRole('combobox', { name: 'Task state' })
        );
        expect(screen.queryByRole('listbox')).toBeNull();
        expect(change).not.toHaveBeenCalled();
    });
});
