import { createContext, useContext, useId, useLayoutEffect } from 'react';
import type { ButtonHTMLAttributes, JSX } from 'react';

export const DialogFormContext = createContext<{
    register: (id: string, dirty: boolean) => void;
    requestClose: () => void;
} | null>(null);

export function DirtyForm({ dirty }: { dirty: boolean }): null {
    const scope = useContext(DialogFormContext);
    const id = useId();
    useLayoutEffect(() => {
        scope?.register(id, dirty);
        return () => scope?.register(id, false);
    }, [dirty, id, scope]);
    return null;
}

export function DialogCloseButton(
    props: ButtonHTMLAttributes<HTMLButtonElement>
): JSX.Element {
    const scope = useContext(DialogFormContext);
    return (
        <button
            {...props}
            onClick={() => scope?.requestClose()}
            type='button'
        />
    );
}
