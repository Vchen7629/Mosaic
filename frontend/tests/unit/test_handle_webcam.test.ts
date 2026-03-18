import { renderHook } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach} from 'vitest'
import { useFaceCapture } from "../../src/hooks/useFaceCapture" 

const mockTrackStop = vi.fn()
const mockStream = {
    getTracks: () => [{ stop: mockTrackStop }],
} as unknown as MediaStream

beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    Object.defineProperty(globalThis.navigator, "mediaDevices", {
        configurable: true,
        value: {
            getUserMedia: vi.fn().mockResolvedValue(mockStream)
        }
    })
    HTMLVideoElement.prototype.play = vi.fn().mockResolvedValue(undefined)
    HTMLCanvasElement.prototype.getContext = vi.fn().mockReturnValue({
        drawImage: vi.fn(),
    })
    HTMLCanvasElement.prototype.toDataURL = vi.fn().mockReturnValue("data:image/jpeg;base64,fakedata")
})

describe("useFaceCapture - webcam open", () => {
    it("opens the webcam with correct constraints when enabled", async () => {
        const onFrame = vi.fn()
        renderHook(() => useFaceCapture({ enabled: true, onFrame }))

        await vi.waitFor(() => {
            expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledWith({ 
                video: { width: 640, height: 480 } 
            })
        })
    })

    it("does not open the webcam when disabled", async () => {
        const onFrame = vi.fn()
        renderHook(() => useFaceCapture({ enabled: false, onFrame }))

        vi.runAllTimers()
        await Promise.resolve()
        expect(navigator.mediaDevices.getUserMedia).not.toHaveBeenCalled()
    })

    it("calls onFrame on the interval once webcam is open", async () => {
        const onFrame = vi.fn()
        renderHook(() => useFaceCapture({ enabled: true, onFrame }))

        await vi.waitFor(() => {
            expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalled()
        })

        vi.advanceTimersByTime(1000)
        expect(onFrame).toHaveBeenCalled()
    })

    it("stops webcam tracks on onmount", async () => {
        const onFrame = vi.fn()
        const { unmount } = renderHook(() => useFaceCapture({ enabled: true, onFrame }))

        await vi.waitFor(() => {
            expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalled()
        })

        unmount()
        expect(mockTrackStop).toHaveBeenCalled()
    })

    it("stops webcam tracks when enabled flips to false", async () => {
        const onFrame = vi.fn()
        const { rerender } = renderHook(
            ({ enabled }: { enabled: boolean }) => useFaceCapture({ enabled, onFrame }),
            { initialProps: { enabled: true } }
        )

        await vi.waitFor(() => {
            expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalled()
        })

        rerender({ enabled: false })
        expect(mockTrackStop).toHaveBeenCalled()
    })

    it("stops orphaned stream tracks if component unmounts before getUserMedia resolves", async () => {
        let resolveStream!: (stream: MediaStream) => void
        navigator.mediaDevices.getUserMedia = vi.fn().mockReturnValue(
            new Promise<MediaStream>(resolve => { resolveStream = resolve })
        )

        const onFrame = vi.fn()
        const { unmount } = renderHook(() => useFaceCapture({ enabled: true, onFrame }))

        unmount()
        resolveStream(mockStream)

        await vi.waitFor(() => {
            expect(mockTrackStop).toHaveBeenCalled()
        })
    })
})