import { useEffect, useRef } from "react"

interface UseFaceCaptureProps {
    enabled: boolean
    onFrame: (frame: string) => void
    fps?: number
    testMode?: boolean  // Enable to save frames to downloads
}

/**
 * 
 * @param enabled
 * @param onFrame
 * @param fps
 */
export function useFaceCapture({ enabled, onFrame, fps = 2, testMode = false}: UseFaceCaptureProps) {
    const videoRef = useRef<HTMLVideoElement>(document.createElement("video"))
    const canvasRef = useRef<HTMLCanvasElement>(document.createElement("canvas"))
    const streamRef = useRef<MediaStream | null>(null)
    const frameCountRef = useRef(0)

    useEffect(() => {
        if (!enabled) return

        let interval: number
        let webCamMounted = true // flag to track if the webcam is opened

        navigator.mediaDevices.getUserMedia({ video: { width: 640, height: 480} })
            .then(stream => {
                if (!webCamMounted) { // prevent webcam from opening if orphaned
                    stream.getTracks().forEach(t => t.stop())
                    return
                }
                streamRef.current = stream
                videoRef.current.srcObject = stream
                videoRef.current.play()

                interval = setInterval(() => {
                    const frame = CaptureFrame(videoRef.current, canvasRef.current)
                    onFrame(frame)

                    // Save to downloads if in test mode
                    if (testMode) {
                        saveFrameToFile(frame, frameCountRef.current)
                        frameCountRef.current++
                    }
                }, 1000 / fps)
            })

        return () => {
            webCamMounted = false
            clearInterval(interval);
            streamRef.current?.getTracks().forEach(t => t.stop())
            streamRef.current = null
            frameCountRef.current = 0
        }
    }, [enabled, fps, testMode, onFrame])
}

/**
 * helper function to draw to canvas and convert the frame to JPEG
 * @param video
 * @param canvas
 * @param quality
 * @returns base64 JPEG without data URL prefix
 */
export function CaptureFrame(
    video: HTMLVideoElement,
    canvas: HTMLCanvasElement,
    quality: number = 0.8
): string {
    canvas.width = video.videoWidth
    canvas.height = video.videoHeight

    const ctx = canvas.getContext("2d")
    if (!ctx) throw new Error("Failed to get canvas context")

    ctx.drawImage(video, 0, 0)

    // base64 JPEG without data URL prefix
    return canvas.toDataURL("image/jpeg", quality).split(",")[1]
}

/**
 * helper function to save frame as JPEG file to downloads
 * @param base64Data
 * @param frameNumber
 */
function saveFrameToFile(base64Data: string, frameNumber: number): void {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-')
    const link = document.createElement('a')
    link.href = `data:image/jpeg;base64,${base64Data}`
    link.download = `face_capture_${timestamp}_frame${frameNumber}.jpg`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
}