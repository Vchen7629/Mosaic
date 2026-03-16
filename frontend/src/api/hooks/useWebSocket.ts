import { RefObject, useEffect, useRef } from "react";

/**
 * hook to handle websocket lifecycle and connecting to the backend client
 * @param patientId 
 * @param isActive 
 * @returns a ref with the websocket connection
 */
export function useWebSocketConnection(isActive: boolean): RefObject<WebSocket | null> {
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