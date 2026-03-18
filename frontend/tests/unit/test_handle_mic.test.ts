import { renderHook } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach} from 'vitest'
import { useAudioCapture } from "../../src/hooks/useAudioCapture" 

const mockTrackStop = vi.fn()
const mockStream = {
    getTracks: () => [{ stop: mockTrackStop }],
} as unknown as MediaStream

const mockAudioContextClose = vi.fn().mockResolvedValue(undefined)
const mockProcessorDisconnect = vi.fn()
const mockConnect = vi.fn()

const mockProcessor = {
    onaudioprocess: null as unknown,
    connect: mockConnect,
    disconnect: mockProcessorDisconnect
}

class MockAudioContext {
    state = "running"
    destination = {}
    close = mockAudioContextClose
    createScriptProcessor = vi.fn().mockReturnValue(mockProcessor)
    createMediaStreamSource = vi.fn().mockReturnValue({ connect: mockConnect })
}

describe("useAudioCapture - mic handling", () => {
    beforeEach(() => {
        vi.clearAllMocks()
        Object.defineProperty(globalThis.navigator, "mediaDevices", {
            configurable: true,
            value: {
                getUserMedia: vi.fn().mockResolvedValue(mockStream)
            }
        })
        globalThis.AudioContext = MockAudioContext as unknown as typeof AudioContext
    })

    it("opens the mic when enabled", async () => {
        const onAudioData = vi.fn()
        renderHook(() => useAudioCapture({ enabled: true, onAudioData }))

        await vi.waitFor(() => {
            expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledWith({ audio: true })
        })
    })

    it("does not open the mic when disabled", async () => {
        const onAudioData = vi.fn()
        renderHook(() => useAudioCapture({ enabled: false, onAudioData }))

        await new Promise((r) => setTimeout(r, 50))
        expect(navigator.mediaDevices.getUserMedia).not.toHaveBeenCalled()
    })

    it("stops mic tracks and closes AudioContext on unmount", async () => {
        const onAudioData = vi.fn()
        const { unmount } = renderHook(() => useAudioCapture({ enabled: true, onAudioData }))

        await vi.waitFor(() => {
            expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalled()
        })

        unmount()

        expect(mockTrackStop).toHaveBeenCalled()
        expect(mockAudioContextClose).toHaveBeenCalled()
        expect(mockProcessorDisconnect).toHaveBeenCalled()
    })

    it("stops mic tracks and closes AudioContext when enabled flips to false", async () => {
        const onAudioData = vi.fn()
        const { rerender } = renderHook(
            ({ enabled }: { enabled: boolean }) => useAudioCapture({ enabled, onAudioData }),
            { initialProps: { enabled: true } }
        )

        await vi.waitFor(() => {
            expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalled()
        })

        rerender({ enabled: false })

        expect(mockTrackStop).toHaveBeenCalled()
        expect(mockAudioContextClose).toHaveBeenCalled()
    })

    it("logs error and does not crash when getUserMedia rejects (e.g. mic permission denied)", async () => {
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {})
        navigator.mediaDevices.getUserMedia = vi.fn().mockRejectedValue(new Error("Permission denied"))

        const onAudioData = vi.fn()
        renderHook(() => useAudioCapture({ enabled: true, onAudioData }))

        await vi.waitFor(() => {
            expect(consoleError).toHaveBeenCalledWith("Audio capture failed:", expect.any(Error))
        })

        consoleError.mockRestore()
    })

    it("calls onAudioData with samples when onaudioprocess fires", async () => {
        const onAudioData = vi.fn()
        renderHook(() => useAudioCapture({ enabled: true, onAudioData }))

        await vi.waitFor(() => {
            expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalled()
        })

        const fakeSamples = new Float32Array([0.1, 0.2, 0.3])
        const fakeEvent = { inputBuffer: { getChannelData: vi.fn().mockReturnValue(fakeSamples) } }
        ;(mockProcessor.onaudioprocess as unknown as (e: typeof fakeEvent) => void)(fakeEvent)

        expect(onAudioData).toHaveBeenCalledWith(fakeSamples)
    })
})