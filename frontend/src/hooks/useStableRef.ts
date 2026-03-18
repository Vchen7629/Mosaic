import { useRef } from "react";

/**
 * Stable ref hook for callbacks inside effects. Don't need to add
 * them to dependency array and prevents unnecessary effect reruns
 * @param value - callback function
 * @returns a mutable ref that is always synced to latest value
 */
export function useStableRef<T>(value: T) {
    const ref = useRef(value)
    ref.current = value
    return ref
}