import { renderHook } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { useStableRef } from "../../src/hooks/useStableRef"

describe("useStableRef", () => {
    it("holds the initial value on mount", () => {
        const { result } = renderHook(() => useStableRef(42))

        expect(result.current.current).toBe(42)
    })

    it("updates ref.current when the value changes on rerender", () => {
        const { result, rerender } = renderHook(
            ({ value }: { value: number }) => useStableRef(value),
            { initialProps: { value: 1 } }
        )

        rerender({ value: 99 })

        expect(result.current.current).toBe(99)
    })

    it("returns the same ref object across rerenders", () => {
        const { result, rerender } = renderHook(
            ({ value }: { value: number }) => useStableRef(value),
            { initialProps: { value: 1 } }
        )

        const refBefore = result.current
        rerender({ value: 2 })

        expect(result.current).toBe(refBefore)
    })

    it("works with callback functions", () => {
        const cb1 = vi.fn()
        const cb2 = vi.fn()

        const { result, rerender } = renderHook(
            ({ cb }: { cb: () => void }) => useStableRef(cb),
            { initialProps: { cb: cb1 } }
        )

        expect(result.current.current).toBe(cb1)

        rerender({ cb: cb2 })

        expect(result.current.current).toBe(cb2)
    })
})
