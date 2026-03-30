import { renderHook, act } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach } from "vitest"
import { useVisitorFace, useNewVisitorFaceRegister, useSyncProfileProcess } from "../../src/api/hooks/face"

let capturedOnFrame: ((frame: string) => void) | null = null

vi.mock("../../src/hooks/useFaceCapture", () => ({
    useFaceCapture: vi.fn(({ onFrame }) => {
        capturedOnFrame = onFrame
    }),
}))

function makeWs(readyState = WebSocket.OPEN) {
    const listeners: Record<string, ((e: MessageEvent) => void)[]> = {}
    const ws = {
        readyState,
        send: vi.fn(),
        addEventListener: vi.fn((event: string, handler: (e: MessageEvent) => void) => {
            if (!listeners[event]) listeners[event] = []
            listeners[event].push(handler)
        }),
        removeEventListener: vi.fn((event: string, handler: (e: MessageEvent) => void) => {
            if (listeners[event]) listeners[event] = listeners[event].filter(h => h !== handler)
        }),
    }
    function dispatch(data: object) {
        listeners["message"]?.forEach(h => h({ data: JSON.stringify(data) } as MessageEvent))
    }
    return { ws: ws as unknown as WebSocket, dispatch }
}

beforeEach(() => {
    vi.clearAllMocks()
    capturedOnFrame = null
    localStorage.clear()
})

// ─── useVisitorFace ───────────────────────────────────────────────────────────

describe("useVisitorFace - message handling", () => {
    it("calls onNewFaceDetected when new_visitor_register message received", () => {
        const { ws, dispatch } = makeWs()
        const onNewFaceDetected = vi.fn()

        renderHook(() => useVisitorFace(ws, true, "profile1", onNewFaceDetected, vi.fn(), vi.fn()))

        dispatch({ type: "new_visitor_register", session_token: "tok_42", face_embedding: [0.1, 0.2] })

        expect(onNewFaceDetected).toHaveBeenCalledWith("tok_42", JSON.stringify([0.1, 0.2]))
    })

    it("calls onExistingVisitorDetected and onBriefingRecieved for visitor_briefing_response", () => {
        const { ws, dispatch } = makeWs()
        const onExistingVisitorDetected = vi.fn()
        const onBriefingRecieved = vi.fn()

        renderHook(() => useVisitorFace(ws, true, "profile1", vi.fn(), onExistingVisitorDetected, onBriefingRecieved))

        dispatch({ type: "visitor_briefing_response", visitor_id: "v1", visitor_name: "Alice", briefing: "Alice is great." })

        expect(onExistingVisitorDetected).toHaveBeenCalledWith("v1")
        expect(onBriefingRecieved).toHaveBeenCalledWith({ visitorName: "Alice", briefingText: "Alice is great." })
    })

    it("deduplicates existing visitors — onExistingVisitorDetected only fires once per visitor", () => {
        const { ws, dispatch } = makeWs()
        const onExistingVisitorDetected = vi.fn()

        renderHook(() => useVisitorFace(ws, true, "profile1", vi.fn(), onExistingVisitorDetected, vi.fn()))

        dispatch({ type: "visitor_briefing_response", visitor_id: "v1", visitor_name: "Alice", briefing: "text" })
        dispatch({ type: "visitor_briefing_response", visitor_id: "v1", visitor_name: "Alice", briefing: "text" })

        expect(onExistingVisitorDetected).toHaveBeenCalledTimes(1)
    })

    it("resets seen visitor IDs when isCapturingFace flips back to true", async () => {
        const { ws, dispatch } = makeWs()
        const onExistingVisitorDetected = vi.fn()

        const { rerender } = renderHook(
            ({ capturing }: { capturing: boolean }) =>
                useVisitorFace(ws, capturing, "profile1", vi.fn(), onExistingVisitorDetected, vi.fn()),
            { initialProps: { capturing: true } }
        )

        dispatch({ type: "visitor_briefing_response", visitor_id: "v1", visitor_name: "Alice", briefing: "text" })
        expect(onExistingVisitorDetected).toHaveBeenCalledTimes(1)

        await act(async () => { rerender({ capturing: false }) })
        await act(async () => { rerender({ capturing: true }) })

        dispatch({ type: "visitor_briefing_response", visitor_id: "v1", visitor_name: "Alice", briefing: "text" })
        expect(onExistingVisitorDetected).toHaveBeenCalledTimes(2)
    })

    it("ignores messages when isCapturingFace is false", () => {
        const { ws, dispatch } = makeWs()
        const onNewFaceDetected = vi.fn()

        renderHook(() => useVisitorFace(ws, false, "profile1", onNewFaceDetected, vi.fn(), vi.fn()))

        dispatch({ type: "new_visitor_register", session_token: "tok_1", face_embedding: [] })

        expect(onNewFaceDetected).not.toHaveBeenCalled()
    })

    it("ignores unknown message types", () => {
        const { ws, dispatch } = makeWs()
        const onNewFaceDetected = vi.fn()
        const onExistingVisitorDetected = vi.fn()

        renderHook(() => useVisitorFace(ws, true, "profile1", onNewFaceDetected, onExistingVisitorDetected, vi.fn()))

        dispatch({ type: "unknown_type", data: "whatever" })

        expect(onNewFaceDetected).not.toHaveBeenCalled()
        expect(onExistingVisitorDetected).not.toHaveBeenCalled()
    })

    it("sends visitor_face frame to ws when onFrame is called and ws is open", () => {
        const { ws } = makeWs()

        renderHook(() => useVisitorFace(ws, true, "profile1", vi.fn(), vi.fn(), vi.fn()))

        capturedOnFrame!("base64frame")

        expect(ws.send).toHaveBeenCalledWith(JSON.stringify({
            type: "visitor_face",
            face_bytes: "base64frame",
            session_token: "profile1",
        }))
    })

    it("does not send frame when ws is closed", () => {
        const { ws } = makeWs(WebSocket.CLOSED)

        renderHook(() => useVisitorFace(ws, true, "profile1", vi.fn(), vi.fn(), vi.fn()))

        capturedOnFrame!("base64frame")

        expect(ws.send).not.toHaveBeenCalled()
    })
})

// ─── useNewVisitorFaceRegister ────────────────────────────────────────────────

describe("useNewVisitorFaceRegister", () => {
    it("sends new_visitor_face when shouldRegister is true and ws is open", () => {
        const { ws } = makeWs()

        renderHook(() => useNewVisitorFaceRegister(ws, true, "emb123", "profile1", "Alice", vi.fn()))

        expect(ws.send).toHaveBeenCalledWith(JSON.stringify({
            type: "new_visitor_face",
            face_embedding: "emb123",
            session_token: "profile1",
            visitor_name: "Alice",
        }))
    })

    it("does not send when shouldRegister is false", () => {
        const { ws } = makeWs()

        renderHook(() => useNewVisitorFaceRegister(ws, false, "emb123", "profile1", "Alice", vi.fn()))

        expect(ws.send).not.toHaveBeenCalled()
    })

    it("does not send when ws is not open", () => {
        const { ws } = makeWs(WebSocket.CLOSED)

        renderHook(() => useNewVisitorFaceRegister(ws, true, "emb123", "profile1", "Alice", vi.fn()))

        expect(ws.send).not.toHaveBeenCalled()
    })

    it("calls onSuccess when register_visitor_resp is received", () => {
        const { ws, dispatch } = makeWs()
        const onSuccess = vi.fn()

        renderHook(() => useNewVisitorFaceRegister(ws, false, "emb123", "profile1", "Alice", onSuccess))

        dispatch({ type: "register_visitor_resp", visitor_id: 99 })

        expect(onSuccess).toHaveBeenCalledWith("99")
    })

    it("ignores messages that are not register_visitor_resp", () => {
        const { ws, dispatch } = makeWs()
        const onSuccess = vi.fn()

        renderHook(() => useNewVisitorFaceRegister(ws, false, "emb123", "profile1", "Alice", onSuccess))

        dispatch({ type: "some_other_type", visitor_id: 1 })

        expect(onSuccess).not.toHaveBeenCalled()
    })
})


describe("useSyncProfileProcess", () => {
    it("sends sync_profile once frameCount frames have been captured", () => {
        const { ws } = makeWs()

        renderHook(() => useSyncProfileProcess(ws, true, vi.fn(), 3))

        capturedOnFrame!("frame1")
        capturedOnFrame!("frame2")
        capturedOnFrame!("frame3")

        expect(ws.send).toHaveBeenCalledWith(JSON.stringify({
            type: "sync_profile",
            frames: ["frame1", "frame2", "frame3"],
        }))
    })

    it("does not send before frameCount is reached", () => {
        const { ws } = makeWs()

        renderHook(() => useSyncProfileProcess(ws, true, vi.fn(), 3))

        capturedOnFrame!("frame1")
        capturedOnFrame!("frame2")

        expect(ws.send).not.toHaveBeenCalled()
    })

    it("only sends once even if more frames arrive after frameCount", () => {
        const { ws } = makeWs()

        renderHook(() => useSyncProfileProcess(ws, true, vi.fn(), 2))

        capturedOnFrame!("frame1")
        capturedOnFrame!("frame2")
        capturedOnFrame!("frame3")

        expect(ws.send).toHaveBeenCalledTimes(1)
    })

    it("resets hasSent when isCapturingFace flips back to true, allowing a new sync", async () => {
        const { ws } = makeWs()

        const { rerender } = renderHook(
            ({ capturing }: { capturing: boolean }) => useSyncProfileProcess(ws, capturing, vi.fn(), 2),
            { initialProps: { capturing: true } }
        )

        capturedOnFrame!("frame1")
        capturedOnFrame!("frame2")
        expect(ws.send).toHaveBeenCalledTimes(1)

        await act(async () => { rerender({ capturing: false }) })
        await act(async () => { rerender({ capturing: true }) })

        capturedOnFrame!("frame1")
        capturedOnFrame!("frame2")
        expect(ws.send).toHaveBeenCalledTimes(2)
    })

    it("calls onProfileSynced and saves session_token to localStorage on profile_face_response", () => {
        const { ws, dispatch } = makeWs()
        const onProfileSynced = vi.fn()

        renderHook(() => useSyncProfileProcess(ws, true, onProfileSynced))

        dispatch({ type: "profile_face_response", session_token: "tok_abc" })

        expect(onProfileSynced).toHaveBeenCalledWith("tok_abc")
        expect(localStorage.getItem("session_token")).toBe("tok_abc")
    })

    it("ignores messages that are not profile_face_response", () => {
        const { ws, dispatch } = makeWs()
        const onProfileSynced = vi.fn()

        renderHook(() => useSyncProfileProcess(ws, true, onProfileSynced))

        dispatch({ type: "other_type", session_token: "tok_abc" })

        expect(onProfileSynced).not.toHaveBeenCalled()
    })

    it("does not send frames when ws is closed", () => {
        const { ws } = makeWs(WebSocket.CLOSED)

        renderHook(() => useSyncProfileProcess(ws, true, vi.fn(), 2))

        capturedOnFrame!("frame1")
        capturedOnFrame!("frame2")

        expect(ws.send).not.toHaveBeenCalled()
    })
})
