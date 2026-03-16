import { useEffect, useRef } from "react";

/**
 * hook to handle websocket lifecycle and connecting to the backend client
 * @param patientId 
 * @param isActive 
 * @returns 
 */
export function useWebSocketConnection(isActive: boolean) {
    const wsRef = useRef<WebSocket | null>(null)

    useEffect(() => {
        if (!isActive) {
            wsRef.current?.close()
            wsRef.current = null
            return
        }

        const url = `ws://localhost:8000/api/v1/ws`
            
        wsRef.current = new WebSocket(url)
        
        return () => wsRef.current?.close()
    }, [isActive])

    return wsRef
}