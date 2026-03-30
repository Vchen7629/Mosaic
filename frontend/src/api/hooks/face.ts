import { useCallback, useEffect, useRef } from "react"
import { useFaceCapture } from "../../hooks/useFaceCapture"
import { BriefingComponent } from "../../types/briefing"
import { useStableRef } from "../../hooks/useStableRef"

/**
 * @description send to images to backend to process faces to detect new visitors
 * @param isCapturingFace boolean to control whether to start the webcam
 */
export function useVisitorFace(
    ws: WebSocket | null,
    isCapturingFace: boolean,
    sessionToken: string | null,
    onNewFaceDetected: (
        sessionToken: number,
        faceEmbedding: string,
    ) => void,
    onExistingVisitorDetected: (visitorID: string) => void,
    onBriefingRecieved: (briefing: BriefingComponent) => void
) {
    const onNewFaceDetectedRef = useStableRef(onNewFaceDetected)
    const onExistingVisitorDetectedRef = useStableRef(onExistingVisitorDetected)
    const onBriefingRecievedRef = useStableRef(onBriefingRecieved)
    const isCapturingFaceRef = useStableRef(isCapturingFace)

    const seenVisitorIdsRef = useRef<Set<string>>(new Set())
    useEffect(() => {
        if (isCapturingFace) seenVisitorIdsRef.current = new Set()
    }, [isCapturingFace])

    // reacts to the ws messages sent from backend to frontend
    useEffect(() => {
        if (!isCapturingFace) return

        if (!ws) return

        const handleMessage = (event: MessageEvent) => {
            if (!isCapturingFaceRef.current) return
            const data = JSON.parse(event.data)
            if (data.type !== "new_visitor_register" && data.type !== "visitor_briefing_response") return

            if (data.type == "new_visitor_register") {
                onNewFaceDetectedRef.current(data.session_token, JSON.stringify(data.face_embedding))
            }
            if (data.type == "visitor_briefing_response") {
                const visitorId = String(data.visitor_id)
                if (seenVisitorIdsRef.current.has(visitorId)) return
                seenVisitorIdsRef.current.add(visitorId)
                console.log("visitor_briefing_response received", data)
                onExistingVisitorDetectedRef.current(visitorId)
                onBriefingRecievedRef.current({ visitorName: data.visitor_name, briefingText: data.briefing })
            }
        }

        ws.addEventListener("message", handleMessage)
        return () => ws.removeEventListener("message", handleMessage)
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [isCapturingFace, ws])

    const onFrame = useCallback(( frame: string ) => {
        if (!ws || ws?.readyState !== WebSocket.OPEN) return

        ws.send(JSON.stringify({
            type: "visitor_face",
            face_bytes: frame,
            session_token: sessionToken
        }))
    }, [ws, sessionToken])

    useFaceCapture({ enabled: isCapturingFace, onFrame, testMode: false })
}


/**
 * 
 */
export function useNewVisitorFaceRegister(
    ws: WebSocket | null, 
    shouldRegister: boolean,
    faceEmbedding: string,
    sessionToken: string | null,
    visitorName: string,
    onSuccess: (visitorId: string) => void
) {
    const onSuccessRef = useStableRef(onSuccess)

    useEffect(() => {
        if (!shouldRegister || !ws || ws.readyState !== WebSocket.OPEN) return

        ws.send(JSON.stringify({
            type: "new_visitor_face",
            face_embedding: faceEmbedding,
            session_token: sessionToken,
            visitor_name: visitorName
        }))
    }, [shouldRegister, ws, faceEmbedding, sessionToken, visitorName])

    useEffect(() => {
        if (!ws) return
        const handleMessage = (event: MessageEvent) => {
            const data = JSON.parse(event.data)
            if (data.type !== "register_visitor_resp") return

            onSuccessRef.current(String(data.visitor_id))
        }

        ws.addEventListener("message", handleMessage)
        return () => ws.removeEventListener("message", handleMessage)
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [ws])
}

/**
 * send to images to backend to process faces to sync user profile,
 * saves the returned profile_id to localStorage and calls onProfileSynced
 * @param isCapturingFace boolean to control whether to start the webcam
 * @param onProfileSynced called with the profile_id once a face is confirmed
 * @param frameCount number of face frame to capture before sending to backend
 */
export function useSyncProfileProcess(
    ws: WebSocket | null, 
    isCapturingFace: boolean,
    onProfileSynced: (sessionToken: string) => void,
    frameCount: number = 5
) {
    const framesRef = useRef<string[]>([])
    const hasSentRef = useRef(false)

    const onProfileSyncedRef = useStableRef(onProfileSynced)

    // reset on start capturingface
    useEffect(() => {
        if (isCapturingFace) hasSentRef.current = false
    }, [isCapturingFace])

    // reacts to the ws messages sent from backend to frontend
    useEffect(() => {
        if (!isCapturingFace) return

        if (!ws) return

        // saves the session_token to localstorage for later use
        const handleMessage = (event: MessageEvent) => {
            const data = JSON.parse(event.data)
            if (data.type !== "profile_face_response") return
            localStorage.setItem("session_token", String(data.session_token))
            onProfileSyncedRef.current(data.session_token)
        }

        ws.addEventListener("message", handleMessage)
        return () => {
            ws.removeEventListener("message", handleMessage)
            framesRef.current = []
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
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