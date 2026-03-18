import { renderHook } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach } from "vitest"
import { useBackendLifecycle } from "../../src/hooks/useBackendLifecycle"

const { mockInvoke, mockOnCloseRequested, mockDestroy, mockUnlisten } = vi.hoisted(() => {
    const mockUnlisten = vi.fn()
    const mockDestroy = vi.fn().mockResolvedValue(undefined)
    const mockOnCloseRequested = vi.fn().mockResolvedValue(mockUnlisten)
    const mockInvoke = vi.fn().mockResolvedValue(undefined)
    return { mockInvoke, mockOnCloseRequested, mockDestroy, mockUnlisten }
})

vi.mock("@tauri-apps/api/core", () => ({
    invoke: mockInvoke,
}))

vi.mock("@tauri-apps/api/window", () => ({
    getCurrentWindow: vi.fn().mockReturnValue({
        onCloseRequested: mockOnCloseRequested,
        destroy: mockDestroy,
    }),
}))

async function simulateCloseRequest() {
    const handler = mockOnCloseRequested.mock.calls[0][0]
    const mockEvent = { preventDefault: vi.fn() }
    await handler(mockEvent)
    return mockEvent
}

beforeEach(() => {
    vi.clearAllMocks()
    mockOnCloseRequested.mockResolvedValue(mockUnlisten)
    mockDestroy.mockResolvedValue(undefined)
    mockInvoke.mockResolvedValue(undefined)
})

describe("useBackendLifecycle - happy path", () => {
    it("invokes start_backend_api on mount", async () => {
        renderHook(() => useBackendLifecycle())

        await vi.waitFor(() => {
            expect(mockInvoke).toHaveBeenCalledWith("start_backend_api")
        })
    })

    it("registers an onCloseRequested listener on mount", () => {
        renderHook(() => useBackendLifecycle())

        expect(mockOnCloseRequested).toHaveBeenCalled()
    })

    it("calls event.preventDefault when close is requested", async () => {
        renderHook(() => useBackendLifecycle())

        await vi.waitFor(() => expect(mockOnCloseRequested).toHaveBeenCalled())
        const mockEvent = await simulateCloseRequest()

        expect(mockEvent.preventDefault).toHaveBeenCalled()
    })

    it("invokes stop_backend_api when close is requested", async () => {
        renderHook(() => useBackendLifecycle())

        await vi.waitFor(() => expect(mockOnCloseRequested).toHaveBeenCalled())
        await simulateCloseRequest()

        expect(mockInvoke).toHaveBeenCalledWith("stop_backend_api")
    })

    it("destroys the window after stopping the backend", async () => {
        renderHook(() => useBackendLifecycle())

        await vi.waitFor(() => expect(mockOnCloseRequested).toHaveBeenCalled())
        await simulateCloseRequest()

        expect(mockDestroy).toHaveBeenCalled()
    })

    it("calls the unlisten function on unmount", async () => {
        const { unmount } = renderHook(() => useBackendLifecycle())

        await vi.waitFor(() => expect(mockOnCloseRequested).toHaveBeenCalled())
        unmount()

        await vi.waitFor(() => expect(mockUnlisten).toHaveBeenCalled())
    })
})

describe("useBackendLifecycle - edge cases", () => {
    it("swallows error if start_backend_api rejects", async () => {
        mockInvoke.mockRejectedValueOnce(new Error("start failed"))

        expect(() => renderHook(() => useBackendLifecycle())).not.toThrow()
    })

    it("swallows error if stop_backend_api rejects during close", async () => {
        mockInvoke
            .mockResolvedValueOnce(undefined)
            .mockRejectedValueOnce(new Error("stop failed"))

        renderHook(() => useBackendLifecycle())

        await vi.waitFor(() => expect(mockOnCloseRequested).toHaveBeenCalled())

        await expect(simulateCloseRequest()).resolves.not.toThrow()
    })
})
