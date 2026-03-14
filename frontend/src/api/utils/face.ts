import { RefObject } from "react";
import { useFaceCapture } from "../../hooks/useFaceCapture";

/**
 * @description handles websocket connection to send face images to backend for processing 
 * @param isRecording 
 */
export function backendFaceProcess(wsRef: RefObject<WebSocket | null>, isRecording: boolean) {
    useFaceCapture({
        enabled: isRecording,
        onFrame: (frame) => {
            if (wsRef.current?.readyState === WebSocket.OPEN) {
                wsRef.current.send(JSON.stringify({
                    type: "face_frame",
                    data: frame,
                }))
            }
        },
        testMode: false
    })
}