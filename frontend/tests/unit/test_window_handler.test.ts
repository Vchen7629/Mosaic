import { renderHook } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach} from 'vitest'
import { useAutoResizeWindow } from "../../src/hooks/useAutoResizeWindow"
import { useSetWindowPosition } from "../../src/hooks/useSetWindowPosition"
import { getCurrentWindow, LogicalPosition } from "@tauri-apps/api/window"

const { mockSetSize, mockSetPosition, mockCurrentWindow, mockCurrentMonitor } = vi.hoisted(() => {
    const mockSetSize = vi.fn()
    const mockSetPosition = vi.fn()
    return {
        mockSetSize,
        mockSetPosition,
        mockCurrentWindow: vi.fn().mockReturnValue({ setSize: mockSetSize, setPosition: mockSetPosition }),
        mockCurrentMonitor: vi.fn(),
    }
})

vi.mock("@tauri-apps/api/window", () => ({
    getCurrentWindow: mockCurrentWindow,
    currentMonitor: mockCurrentMonitor,
    LogicalSize: class LogicalSize {
        constructor(public width: number, public height: number) {}
    },
    LogicalPosition: class LogicalPosition {
        constructor(public x: number, public y: number) {}
    }
}))

const mockObserve = vi.fn()
const mockDisconnect = vi.fn()
const mockResizeObserver = vi.fn().mockImplementation((cb: ResizeObserverCallback ) => ({
    observe: mockObserve,
    disconnect: mockDisconnect,
    _cb: cb
}))

beforeEach(() => {
    vi.clearAllMocks()
    mockSetSize.mockResolvedValue(undefined)
    mockSetPosition.mockResolvedValue(undefined)
    globalThis.ResizeObserver = mockResizeObserver
})

describe("AutoResizeWindow tests", () => {
    it("calls setSize with width 360 and body scrollHeight on mount", async () => {
        Object.defineProperty(document.body, "scrollHeight", { configurable: true, value: 500 })
    
        renderHook(() => useAutoResizeWindow())

        await vi.waitFor(() => {
            expect(mockSetSize).toHaveBeenCalledWith(expect.objectContaining({ width: 360, height: 500 }))
        }) 
    })
    
    it("observes document.body with a ResizeObserver", () => {
        renderHook(() => useAutoResizeWindow())

        expect(mockObserve).toHaveBeenCalledWith(document.body)
    })
    
    it("disconnects the resizeObserver on unmount", () => {
        const { unmount } = renderHook(() => useAutoResizeWindow())
        
        unmount()

        expect(mockDisconnect).toHaveBeenCalled()
    })
})

describe("SetWindowPosition tests", () => {
    it("positions the window at the top-right based on monitor width", async () => {
        mockCurrentMonitor.mockResolvedValue({
            size: { width: 2560 },
            scaleFactor: 2
        })

        renderHook(() => useSetWindowPosition())

        await vi.waitFor(() => {
            expect(mockSetPosition).toHaveBeenCalledWith(expect.objectContaining({ x: 920, y: 0 }))
        }) 
    })
    
    it("does nothing if no monitor is found", async () => {
        mockCurrentMonitor.mockResolvedValue(null)

        renderHook(() => useSetWindowPosition())

        await new Promise((r) => setTimeout(r, 50))
        expect(mockSetPosition).not.toHaveBeenCalled()
    })
})