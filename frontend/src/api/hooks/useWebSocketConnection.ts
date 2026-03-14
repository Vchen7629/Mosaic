import { useEffect, useRef } from "react";


export function useWebSocketConnection(patientId: string | null, isRecording: boolean) {
    const wsRef = useRef<WebSocket | null>(null)

    useEffect(() => {
        if (!isRecording) {
            wsRef.current?.close()
            wsRef.current = null
            return
        }

        wsRef.current = new WebSocket(`ws://localhost:8000/api/v1/ws?patient_id=${patientId}`)
        return () => wsRef.current?.close()
    }, [isRecording, patientId])

    return wsRef
}