import { useEffect, useRef, useState } from "react";

type FaceEvent = 
    | { type: "known_face"; name: string }
    | { type: "unknown_face" }
    | { type: "face_added"; name: string }

export function useFaceDetection(patientId: string | null, isRecording: boolean) {
    const [detectedName, setDetectedName] = useState<string | null>(null)
    const [unknownFaceDetected, setUnknownFaceDetected] = useState(false)
    const wsRef = useRef<WebSocket | null>(null)

    useEffect(() => {
        if (!isRecording || !patientId) return

        const ws = new WebSocket(`ws://localhost:8000/face/detection?patient_id=${patientId}`)
        wsRef.current = ws

        ws.onmessage = (event) => {
            const data: FaceEvent = JSON.parse(event.data)
            if (data.type === "known_face") {
                setDetectedName(data.name)
                setUnknownFaceDetected(false)
            } else if (data.type === "unknown_face") {
                setUnknownFaceDetected(true)
            } else if (data.type === "face_added") {
                setDetectedName(data.name)
                setUnknownFaceDetected(false)
            }
        }

        ws.onerror = (err) => console.error("Face detection WS error:", err)

        return () => {
            ws.close()
            wsRef.current = null
            setDetectedName(null)
            setUnknownFaceDetected(false)
        }
    }, [isRecording, patientId])

    const confirmNewFace = (name: string) => {
        wsRef.current?.send(JSON.stringify({ action: "add_face", name }))
    }

    return {detectedName, unknownFaceDetected, confirmNewFace}
}

