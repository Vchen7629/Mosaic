import { Dispatch, RefObject, SetStateAction, useCallback, useEffect, useRef } from "react";
import { useFaceCapture } from "../../hooks/useFaceCapture";
import { BriefingComponent } from "../../types/briefing";


/**
 * @description send to images to backend to process faces to detect new visitors
 * @param isCapturingFace boolean to control whether to start the webcam
 */
export function VisitorFaceProcess(
    wsRef: RefObject<WebSocket | null>, 
    isCapturingFace: boolean,
    patientID: string | null,
    onNewFaceDetected: (
        patientId: number,
        faceEmbedding: number,
    ) => void,
    setBriefingList: Dispatch<SetStateAction<BriefingComponent[]>>
) {
    // reacts to the ws messages sent from backend to frontend
    useEffect(() => {
        if (!isCapturingFace) return

        const ws = wsRef.current
        if (!ws) return

        const handleMessage = (event: MessageEvent) => {
            const data = JSON.parse(event.data)
            if (data.type !== "new_visitor_register" && data.type !== "existing_visitor_response") return

            if (data.type == "new_visitor_register") {
                onNewFaceDetected(data.patient_id, data.face_embedding)
            }
            if (data.type == "existing_visitor_response") {
                setBriefingList((prev: BriefingComponent[]) => [...prev, { 
                    visitorName: data.visitor_name, 
                    briefingText: data.briefing 
                }])
            }
        }

        ws.addEventListener("message", handleMessage)
        return () => ws.removeEventListener("message", handleMessage)
    }, [isCapturingFace, wsRef.current])

    const onFrame = useCallback(( frame: string ) => {
        if (!wsRef.current || wsRef.current?.readyState !== WebSocket.OPEN) return

        wsRef.current.send(JSON.stringify({
            type: "visitor_face",
            face_bytes: frame,
            patient_id: patientID
        }))
    }, [wsRef, patientID])

    useFaceCapture({ enabled: isCapturingFace, onFrame, testMode: false })
}

export function NewVisitorFaceRegister(
    wsRef: RefObject<WebSocket | null>, 
    faceEmbedding: string,
    patientID: string,
    visitorName: string,
) {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({
            type: "new_visitor_face",
            face_embedding: faceEmbedding,
            patient_id: patientID,
            visitor_name: visitorName
        }))
    }
}


/**
 * @description send to images to backend to process faces to sync user profile,
 * saves the returned patient_id to localStorage and calls onPatientSynced
 * @param isCapturingFace boolean to control whether to start the webcam
 * @param onPatientSynced called with the patient_id once a face is confirmed
 * @param frameCount number of face frame to capture before sending to backend
 */
export function SyncProfileProcess(
    wsRef: RefObject<WebSocket | null>, 
    isCapturingFace: boolean,
    onPatientSynced: (patientId: number) => void,
    frameCount: number = 5
) {
    const framesRef = useRef<string[]>([])

    // reacts to the ws messages sent from backend to frontend
    useEffect(() => {
        if (!isCapturingFace) return

        const ws = wsRef.current
        if (!ws) return

        // saves the patient_id to localstorage for later use
        const handleMessage = (event: MessageEvent) => {
            const data = JSON.parse(event.data)
            if (data.type !== "profile_face_response" || !data.face_detected) return
            localStorage.setItem("patient_id", String(data.patient_id))
            onPatientSynced(data.patient_id)
        }

        ws.addEventListener("message", handleMessage)
        return () => {
            ws.removeEventListener("message", handleMessage)
            framesRef.current = []
        }
    }, [isCapturingFace, wsRef.current])

    const onFrame = useCallback((frame: string) => {
        if (!wsRef.current || wsRef.current?.readyState !== WebSocket.OPEN) return

        framesRef.current.push(frame)
        
        if (framesRef.current.length >= frameCount) {
            wsRef.current.send(JSON.stringify({
                type: "profile_face",
                face_bytes: framesRef.current,
            }))
            framesRef.current = []
        }
    }, [wsRef, frameCount])

    useFaceCapture({ enabled: isCapturingFace, onFrame, testMode: false })
}