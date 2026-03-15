import { RefObject } from "react";
import { useFaceCapture } from "../../hooks/useFaceCapture";

/**
 * @description handles websocket connection to send face images to backend for processing 
 * @param isCapturingFace boolean to control whether to start the webcam
 */
export function backendFaceProcess(wsRef: RefObject<WebSocket | null>, isCapturingFace: boolean) {
    useFaceCapture({
        enabled: isCapturingFace,
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