import { renderHook } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach } from "vitest"
import { useBackendAudioProcess } from "../../src/api/hooks/audio"

vi.mock("../../src/hooks/useAudioCapture", () => ({
    useAudioCapture: vi.fn(({ onAudioData }) => {
        capturedOnAudioData = onAudioData
    }),
}))

let capturedOnAudioData: ((samples: Float32Array) => void) | null = null

beforeEach(() => {
    vi.clearAllMocks()
    capturedOnAudioData = null
})

function makeOpenWs() {
    return { readyState: WebSocket.OPEN, send: vi.fn() } as unknown as WebSocket
}

describe("useBackendAudioProcess - onAudioData callback", () => {
    it("sends encoded audio over websocket when ws is open", () => {
        const ws = makeOpenWs()
        renderHook(() => useBackendAudioProcess(ws, true, "profile1"))

        const samples = new Float32Array([0.1, 0.2, 0.3])
        capturedOnAudioData!(samples)

        expect(ws.send).toHaveBeenCalledOnce()
        const payload = JSON.parse((ws.send as ReturnType<typeof vi.fn>).mock.calls[0][0])
        expect(payload.type).toBe("audio")
        expect(payload.profile_id).toBe("profile1")
        expect(typeof payload.audio_data).toBe("string")
    })

    it("does nothing when ws is null", () => {
        renderHook(() => useBackendAudioProcess(null, true, "profile1"))

        const samples = new Float32Array([0.1, 0.2])
        expect(() => capturedOnAudioData!(samples)).not.toThrow()
    })

    it("does nothing when ws is not open", () => {
        const ws = { readyState: WebSocket.CLOSED, send: vi.fn() } as unknown as WebSocket
        renderHook(() => useBackendAudioProcess(ws, true, "profile1"))

        capturedOnAudioData!(new Float32Array([0.1]))

        expect(ws.send).not.toHaveBeenCalled()
    })

    it("includes the correct profile_id in the payload", () => {
        const ws = makeOpenWs()
        renderHook(() => useBackendAudioProcess(ws, true, "my-profile"))

        capturedOnAudioData!(new Float32Array([0.5]))

        const payload = JSON.parse((ws.send as ReturnType<typeof vi.fn>).mock.calls[0][0])
        expect(payload.profile_id).toBe("my-profile")
    })
})
