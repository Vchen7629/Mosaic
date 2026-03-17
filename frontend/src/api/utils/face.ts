import { Dispatch, RefObject, SetStateAction, useCallback, useEffect, useRef } from "react";
import { useFaceCapture } from "../../hooks/useFaceCapture";
import { BriefingComponent } from "../../types/briefing";


/**
 * @description send to images to backend to process faces to detect new visitors
 * @param isCapturingFace boolean to control whether to start the webcam
 */
export function VisitorFaceProcess(
    ws: WebSocket | null,
    isCapturingFace: boolean,
    profileID: string | null,
    onNewFaceDetected: (
        profileID: number,
        faceEmbedding: string,
    ) => void,
    setBriefingList: Dispatch<SetStateAction<BriefingComponent[]>>
) {
    const onNewFaceDetectedRef = useRef(onNewFaceDetected)
    useEffect(() => { onNewFaceDetectedRef.current = onNewFaceDetected })

    const setBriefingListRef = useRef(setBriefingList)
    useEffect(() => { setBriefingListRef.current = setBriefingList })

    // reacts to the ws messages sent from backend to frontend
    useEffect(() => {
        if (!isCapturingFace) return

        if (!ws) return

        const handleMessage = (event: MessageEvent) => {
            const data = JSON.parse(event.data)
            if (data.type !== "new_visitor_register" && data.type !== "existing_visitor_response") return

            if (data.type == "new_visitor_register") {
                onNewFaceDetectedRef.current(data.profile_id, JSON.stringify(data.face_embedding))
            }
            if (data.type == "existing_visitor_response") {
                setBriefingListRef.current((prev: BriefingComponent[]) => [...prev, {
                    visitorName: data.visitor_name,
                    briefingText: data.briefing
                }])
            }
        }

        ws.addEventListener("message", handleMessage)
        return () => ws.removeEventListener("message", handleMessage)
    }, [isCapturingFace, ws])

    const onFrame = useCallback(( frame: string ) => {
        if (!ws || ws?.readyState !== WebSocket.OPEN) return

        ws.send(JSON.stringify({
            type: "visitor_face",
            face_bytes: frame,
            profile_id: profileID
        }))
    }, [ws, profileID])

    useFaceCapture({ enabled: isCapturingFace, onFrame, testMode: false })
}

export function NewVisitorFaceRegister(
    ws: WebSocket | null, 
    shouldRegister: boolean,
    faceEmbedding: string,
    profileId: string | null,
    visitorName: string,
    onSuccess: (visitorId: string) => void
) {
    const onSuccessRef = useRef(onSuccess)
    onSuccessRef.current = onSuccess

    useEffect(() => {
        if (!shouldRegister || !ws || ws.readyState !== WebSocket.OPEN) return

        ws.send(JSON.stringify({
            type: "new_visitor_face",
            face_embedding: faceEmbedding,
            profile_id: profileId,
            visitor_name: visitorName
        }))
    }, [shouldRegister, ws, faceEmbedding, profileId, visitorName])

    useEffect(() => {
        if (!ws) return
        const handleMessage = (event: MessageEvent) => {
            const data = JSON.parse(event.data)
            if (data.type !== "register_visitor_resp") return

            onSuccessRef.current(String(data.visitor_id))
        }

        ws.addEventListener("message", handleMessage)
        return () => ws.removeEventListener("message", handleMessage)
    }, [ws])
}


/**
 * @description send to images to backend to process faces to sync user profile,
 * saves the returned profile_id to localStorage and calls onProfileSynced
 * @param isCapturingFace boolean to control whether to start the webcam
 * @param onProfileSynced called with the profile_id once a face is confirmed
 * @param frameCount number of face frame to capture before sending to backend
 */
export function SyncProfileProcess(
    ws: WebSocket | null, 
    isCapturingFace: boolean,
    onProfileSynced: (profileId: number) => void,
    frameCount: number = 5
) {
    const framesRef = useRef<string[]>([])
    const hasSentRef = useRef(false)

    const onProfileSyncedRef = useRef(onProfileSynced)
    useEffect(() => { onProfileSyncedRef.current = onProfileSynced })

    // reset on start capturingface
    useEffect(() => {
        if (isCapturingFace) hasSentRef.current = false
    }, [isCapturingFace])

    // reacts to the ws messages sent from backend to frontend
    useEffect(() => {
        if (!isCapturingFace) return

        if (!ws) return

        // saves the profile_id to localstorage for later use
        const handleMessage = (event: MessageEvent) => {
            const data = JSON.parse(event.data)
            if (data.type !== "profile_face_response" || data.type !== "profile_face_response") return
            localStorage.setItem("profile_id", String(data.profile_id))
            onProfileSyncedRef.current(data.profile_id)
        }

        ws.addEventListener("message", handleMessage)
        return () => {
            ws.removeEventListener("message", handleMessage)
            framesRef.current = []
        }
    }, [isCapturingFace, ws])

    const onFrame = useCallback((frame: string) => {
        if (!ws || ws?.readyState !== WebSocket.OPEN) return
        if (hasSentRef.current) return // prevent duplicate send

        framesRef.current.push(frame)
        
        if (framesRef.current.length >= frameCount) {
            ws.send(JSON.stringify({type: "sync_profile", frames: framesRef.current }))
            framesRef.current = []
            hasSentRef.current = true
        }
    }, [ws, frameCount])

    useFaceCapture({ enabled: isCapturingFace, onFrame, testMode: false })
}