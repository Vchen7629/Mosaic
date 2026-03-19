import { describe, it, expect, vi, beforeEach} from 'vitest'
import { CaptureFrame } from "../../src/hooks/useFaceCapture" 

const mockDrawImage = vi.fn()
const mockGetContext = vi.fn().mockReturnValue({ drawImage: mockDrawImage })

beforeEach(() => {
    vi.clearAllMocks()
    HTMLCanvasElement.prototype.getContext = mockGetContext
    HTMLCanvasElement.prototype.toDataURL = vi.fn().mockReturnValue("data:image/jpeg;base64,fakedata")
})

describe("CaptureFrame - JPEG Create", () => {
    it("returns base64 string without the data URL prefix", () => {
        const video = document.createElement("video")
        const canvas = document.createElement("canvas")

        const result = CaptureFrame(video, canvas)

        expect(result).toBe("fakedata")
    })

    it("sets canvas dimensions from video dimensions", () => {
        const video = document.createElement("video")
        const canvas = document.createElement("canvas")
        Object.defineProperty(video, "videoWidth", { value: 640 })
        Object.defineProperty(video, "videoHeight", { value: 480 })
    
        CaptureFrame(video, canvas)

        expect(canvas.width).toBe(640)
        expect(canvas.height).toBe(480)
    })
    
    it("draws the video frame onto the canvas", () => {
        const video = document.createElement("video")
        const canvas = document.createElement("canvas")
    
        CaptureFrame(video, canvas)

        expect(mockDrawImage).toHaveBeenCalledWith(video, 0, 0)
    })
    
    it("throws if canvas context is unavailable", () => {
        HTMLCanvasElement.prototype.getContext = vi.fn().mockReturnValue(null)
        const video = document.createElement("video")
        const canvas = document.createElement("canvas")

        expect(() => CaptureFrame(video, canvas)).toThrow("Failed to get canvas context")
    })
    
})