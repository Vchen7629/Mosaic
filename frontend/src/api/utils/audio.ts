import { useEffect, useRef } from "react";
import { useAudioCapture } from "../../hooks/useAudioCapture";

/**
 * @description handles websocket connection to send audio to backend for processing 
 * @param patientId 
 * @param isRecording 
 */
export function backendAudioProcess(patientId: string | null, isRecording: boolean) {
    const wsRef = useRef<WebSocket | null>(null)

    useEffect(() => {
        if (!isRecording) {
            wsRef.current?.close()
            wsRef.current = null
            return
        }

        const ws = new WebSocket(`ws://localhost:8000/api/v1/ws?patient_id=${patientId}`)
        wsRef.current = ws

        ws.onopen = () => console.log("Audio WebSocket connected")
        ws.onerror = (err) => console.error("Audio WS error:", err)
        ws.onclose = () => console.log("Audio WebSocket closed")

        return () => ws.close()
    }, [isRecording, patientId])

    useAudioCapture({
        enabled: isRecording,
        onAudioData: (samples) => {
            if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
            // encode float32Array with base64
            const base64 = btoa(String.fromCharCode(...new Uint8Array(samples.buffer)))
            console.log(`[Audio Capture] Sending...`)
            wsRef.current.send(JSON.stringify({ type: "audio", data: base64}))
            console.log(`[Audio Capture] Sent websocket`)
        }
    })
}