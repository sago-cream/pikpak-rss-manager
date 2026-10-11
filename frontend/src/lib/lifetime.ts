import { useEffect, useRef } from 'react';

// Mutations may finish after a dialog closes. Their results must not close a newer dialog.
export function useLifetime(): { current: boolean } {
    const active = useRef(true);
    useEffect(() => {
        active.current = true;
        return () => {
            active.current = false;
        };
    }, []);
    return active;
}
