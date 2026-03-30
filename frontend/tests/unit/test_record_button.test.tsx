import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach} from 'vitest'
import { RecordButton } from "../../src/components/recordButton"
import type { SyncState } from "../../src/types/profle"

vi.mock("../../src/api/hooks/face", () => ({
    useSyncProfileProcess: vi.fn()
}))

const defaultProps = {
  ws: null,
  syncState: "idle" as SyncState,
  isRecording: false,
  sessionToken: "profile123",
  visitorIds: [],
  onSyncStart: vi.fn(),
  onSyncCancel: vi.fn(),
  onSyncComplete: vi.fn(),
  onRecordingStart: vi.fn(),
  onRecordingStop: vi.fn()
};

beforeEach(() => {
    vi.clearAllMocks()
})

describe("RecordButton - conditional rendering", () => {
    it("renders 'Sync Profile' button when syncState is idle", () => {
        render(<RecordButton {...defaultProps} syncState="idle"/>)
        expect(screen.getByText("Sync Profile")).toBeTruthy
    })
    
    it("renders 'Cancel' button when syncState is scanning", () => {
        render(<RecordButton {...defaultProps} syncState="scanning"/>)
        expect(screen.getByText("Cancel")).toBeTruthy
    })

    it("renders 'Start' button when synced and not recording", () => {
        render(<RecordButton {...defaultProps} syncState="active" isRecording={false}/>)
        expect(screen.getByText("Start")).toBeTruthy
    })

    it("renders 'Stop' button when synced and recording", () => {
        render(<RecordButton {...defaultProps} syncState="active" isRecording={true}/>)
        expect(screen.getByText("Stop")).toBeTruthy
    })
})

describe("RecordButton - callbacks", () => {
    it("calls onSyncStart when Sync Profile is clicked", () => {
        const onSyncStart = vi.fn()
        render(<RecordButton {...defaultProps} syncState="idle" onSyncStart={onSyncStart}/>)
        fireEvent.click(screen.getByText("Sync Profile"))
        expect(onSyncStart).toHaveBeenCalled()
    })

    it("calls onSyncCancel when Cancel is clicked", () => {
        const onSyncCancel = vi.fn()
        render(<RecordButton {...defaultProps} syncState="scanning" onSyncCancel={onSyncCancel}/>)
        fireEvent.click(screen.getByText("Cancel"))
        expect(onSyncCancel).toHaveBeenCalled()
    })

    it("calls onRecordingStart when Start is clicked", () => {
        const onRecordingStart = vi.fn()
        render(<RecordButton {...defaultProps} syncState="active" isRecording={false} onRecordingStart={onRecordingStart}/>)
        fireEvent.click(screen.getByText("Start"))
        expect(onRecordingStart).toHaveBeenCalled()
    })

    it("calls onRecordingStop when Stop is clicked", () => {
        const onRecordingStop = vi.fn()
        render(<RecordButton {...defaultProps} syncState="active" isRecording={true} onRecordingStop={onRecordingStop}/>)
        fireEvent.click(screen.getByText("Stop"))
        expect(onRecordingStop).toHaveBeenCalled()
    })
})

describe("RecordButton - Websocket message on stop", () => {
    it("sends save_audio_transcript when Stop is clicked and ws is open", () => {                                                                                           
        const mockSend = vi.fn()                                                                                                                                            
        const mockWs = { readyState: WebSocket.OPEN, send: mockSend } as unknown as WebSocket                                                                               

        render(<RecordButton                                                                                                                                                
            {...defaultProps}                                                                                                                                               
            ws={mockWs}                                                                                                                                                     
            syncState="active"                                                                                                                                              
            isRecording={true}                                                                                                                                              
            sessionToken="profile123"
            visitorIds={["visitor1"]}
        />)
        fireEvent.click(screen.getByText("Stop"))

        expect(mockSend).toHaveBeenCalledWith(JSON.stringify({
            type: "save_audio_transcript",
            session_token: "profile123",
            visitor_id_list: ["visitor1"],
        }))                                                                                                                                                                 
    })

    it("does not send a message when ws is closed", () => {
        const mockSend = vi.fn()
        const mockWs = { readyState: WebSocket.CLOSED, send: mockSend } as unknown as WebSocket

        render(<RecordButton 
            {...defaultProps}
            ws={mockWs} 
            syncState="active"
            isRecording={true}
        />)
        fireEvent.click(screen.getByText("Stop"))

        expect(mockSend).not.toHaveBeenCalled()
    })
})

