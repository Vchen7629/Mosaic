import { useEffect, useState } from "react";

/**
 * hook to handle websocket lifecycle and connecting to the backend client
 * @param isActive
 * @returns the websocket instance or null
 */
export function useWebSocketConnection(isActive: boolean): WebSocket | null {
    const [ws, setWs] = useState<WebSocket | null>(null)

    useEffect(() => {
        if (!isActive) {
            setWs(prev => { prev?.close(); return null })
            return
        }

        const socket = new WebSocket(`ws://localhost:8080/api/v1/ws`)
        setWs(socket)

        return () => { socket.close(); setWs(null) }
    }, [isActive])

    return ws
}
