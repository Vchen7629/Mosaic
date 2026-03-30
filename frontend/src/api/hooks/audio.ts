import { useCallback } from "react";
import { useAudioCapture } from "../../hooks/useAudioCapture";

/**
 * @description custom hook to handle websocket connection to send audio to backend for processing 
 * @param sessionToken the synced profile id used for 
 * @param isRecording 
 */
export function useBackendAudioProcess(
    ws: WebSocket | null, 
    isRecording: boolean,
    sessionToken: string | null
) {
    const onAudioData = useCallback((samples: Float32Array) => {
        if (!ws || ws.readyState !== WebSocket.OPEN) return

        const bytes = new Uint8Array(samples.buffer)
        let binary = ""
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i])
        }

        const base64 = btoa(binary)
        console.log(`[Audio Capture] Sending...`)
        ws.send(JSON.stringify({ 
            type: "audio", audio_data: base64, session_token: sessionToken
        }))
        console.log(`[Audio Capture] Sent websocket`)
    }, [ws, sessionToken])

    useAudioCapture({ enabled: isRecording, onAudioData })
}