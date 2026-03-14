import { RefObject } from "react";
import { useAudioCapture } from "../../hooks/useAudioCapture";

/**
 * @description handles websocket connection to send audio to backend for processing 
 * @param patientId 
 * @param isRecording 
 */
export function backendAudioProcess(wsRef: RefObject<WebSocket | null>, isRecording: boolean) {
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