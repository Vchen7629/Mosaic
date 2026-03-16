import { useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import StartRecordingButton from "./components/startRecordingBtn";
import { SetWindowPosition } from "./utils/setWindowPosition";
import { handleBackendLifecycle } from "./utils/handleBackendLifecycle";
import { AutoResizeWindow } from "./utils/autoResizeWindow";
import ConvoBriefingDisplay from "./components/convo_briefing";
import { backendAudioProcess } from "./api/utils/audio";
import { VisitorFaceProcess, SyncProfileProcess } from "./api/utils/face";
import { useWebSocketConnection } from "./api/hooks/useWebSocket";
import { ProfileStatus, SyncProfileButton } from "./components/syncProfile";
import "./App.css";
import { BriefingComponent } from "./types/briefing";

type SyncState = "idle" | "scanning" | "active";

function App() {
  const [isRecording, setIsRecording] = useState(false);
  const [isFaceCapture, setIsFaceCapture] = useState<boolean>(false);
  const [isSyncProfile, setIsSyncProfile] = useState<boolean>(false);
  const patientId = localStorage.getItem("patient_id");
  const patientName = localStorage.getItem("patient_name");
  const [syncState, setSyncState] = useState<SyncState>(patientName ? "active" : "idle");
  const wsRef = useWebSocketConnection(isRecording || isSyncProfile)
  const [briefingList, setBriefingList] = useState<BriefingComponent[]>([])
  const [newFaceDetected, setNewFaceDetected] = useState<boolean>(false)

  SetWindowPosition()
  AutoResizeWindow()
  handleBackendLifecycle()
  backendAudioProcess(wsRef, isRecording, patientId)
  VisitorFaceProcess(
    wsRef, 
    isFaceCapture, 
    patientId,
    (_patientID, _faceEmbedding) => { setNewFaceDetected(true) },
    setBriefingList
  )
  SyncProfileProcess(wsRef, isSyncProfile, (_patientId) => {
    setSyncState("active")
    setIsSyncProfile(false)
  })

  return (
    <main className="w-full p-2">
      {/* Glossy dark panel */}
      <div className="relative flex flex-col rounded-2xl overflow-hidden bg-zinc-900/90 backdrop-blur-2xl border border-white/6">
        {/* Gloss sheen overlay */}
        <div className="absolute inset-x-0 top-0 h-1/2 bg-linear-to-b from-white/5 to-transparent pointer-events-none z-0 rounded-t-2xl" />

        {/* Header */}
        <div className="relative z-10 flex items-center justify-between px-4 py-3 shrink-0">
          <span className="text-[15px] font-medium text-zinc-300 tracking-tight">
            Live Transcription
          </span>
          <div className="flex items-center gap-2">
            <StartRecordingButton 
              isRecording={isRecording} 
              setIsRecording={setIsRecording}
              isCapturingFace={isFaceCapture}
              setIsCapturingFace={setIsFaceCapture}/>
            <button
              onClick={() => getCurrentWindow().close().catch(console.error)}
              className="w-6 h-6 rounded-full flex items-center justify-center text-zinc-500 hover:text-zinc-200 hover:bg-white/8 transition-all duration-150 cursor-pointer"
              aria-label="Close"
            >
              <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                <path d="M1 1l8 8M9 1L1 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
              </svg>
            </button>
          </div>
        </div>

        {/* Divider */}
        <div className="relative z-10 h-px bg-zinc-800 shrink-0" />

        {/* Patient status */}
        <div className="relative z-10 flex items-center px-4 pt-2.5 pb-3 justify-between">
            <ProfileStatus syncState={syncState}/>
            <SyncProfileButton syncState={syncState} setSyncState={setSyncState} setIsCapturingFace={setIsSyncProfile}/>
        </div>
      </div>

      {briefingList.map((briefing: BriefingComponent, i) => (
        <div 
          key={i}
          className={`transition-[opacity,transform] duration-700 ease-out starting:opacity-0 starting:translate-y-3 
                  ${isRecording ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'}`}
        >
          <ConvoBriefingDisplay Name={briefing.visitorName} Briefing={briefing.briefingText}/>
        </div>
      ))}
    </main>
  );
}



export default App;
