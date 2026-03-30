import { renderHook, act } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach} from 'vitest'
import { useWebSocketConnection } from "../../src/api/hooks/useWebSocket" 

const mockClose = vi.fn()
const mockWebSocket = vi.fn().mockImplementation(() => ({ close: mockClose}))

beforeEach(() => {
    vi.clearAllMocks()
    globalThis.WebSocket = mockWebSocket as unknown as typeof WebSocket
})

describe("useWebSocketConnection - websocket handling", () => {
    it("Opens a websocket to the correct URL when active", () => {
        renderHook(() => useWebSocketConnection(true))

        expect(mockWebSocket).toHaveBeenCalledWith("ws://localhost:8080/api/v1/ws")
    })

    it("does not open a websocket when inactive", () => {
        renderHook(() => useWebSocketConnection(false))

        expect(mockWebSocket).not.toHaveBeenCalled()
    })

    it("returns the websocket instance when active", () => {
        const { result } = renderHook(() => useWebSocketConnection(true))

        expect(result.current).not.toBeNull()
    })

    it("returns null when inactive", () => {
        const { result } = renderHook(() => useWebSocketConnection(false))

        expect(result.current).toBeNull()
    })

    it("closes the websocket when isActive flips to false", async () => {
         const { rerender } = renderHook(                                                                                                                                     
            ({ isActive }: { isActive: boolean }) => useWebSocketConnection(isActive),                                                                                       
            { initialProps: { isActive: true } }                                                                                                                             
        )  

        await act(async () => { rerender({ isActive: false })})

        expect(mockClose).toHaveBeenCalled()
    })

    it("closes the websocket on unmount", () => {
        const { unmount } = renderHook(() => useWebSocketConnection(true))

        unmount()

        expect(mockClose).toHaveBeenCalled()
    })
})