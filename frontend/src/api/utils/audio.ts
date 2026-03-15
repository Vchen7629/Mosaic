import { RefObject, useCallback } from "react";
import { useAudioCapture } from "../../hooks/useAudioCapture";

/**
 * @description handles websocket connection to send audio to backend for processing 
 * @param patientId 
 * @param isRecording 
 */
export function backendAudioProcess(
    wsRef: RefObject<WebSocket | null>, 
    isRecording: boolean,
    patientID: string
) {
    const onAudioData = useCallback((samples: Float32Array) => {
        if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return

        const bytes = new Uint8Array(samples.buffer)
        let binary = ""
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i])
        }

        const base64 = btoa(binary)
        console.log(`[Audio Capture] Sending...`)
        wsRef.current.send(JSON.stringify({ 
            type: "audio", audio_data: base64, patient_id: patientID
        }))
        console.log(`[Audio Capture] Sent websocket`)
    }, [wsRef, patientID])

    useAudioCapture({ enabled: isRecording, onAudioData })
}