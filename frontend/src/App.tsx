import { useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { SetWindowPosition } from "./utils/setWindowPosition";
import { useBackendLifecycle } from "./utils/handleBackendLifecycle";
import { AutoResizeWindow } from "./utils/autoResizeWindow";
import ConvoBriefingDisplay from "./components/convo_briefing";
import { useBackendAudioProcess } from "./api/hooks/audio";
import { VisitorFaceProcess } from "./api/utils/face";
import { useWebSocketConnection } from "./api/hooks/useWebSocket";
import { ProfileStatus } from "./components/profileStatus";
import "./App.css";
import { BriefingComponent } from "./types/briefing";
import { SyncState } from "./types/profle";
import { RecordButton } from "./components/recordButton";
import { X } from "lucide-react";

function App() {
  const [isRecording, setIsRecording] = useState(false);
  const [isFaceCapture, setIsFaceCapture] = useState<boolean>(false);
  const [isSyncProfile, setIsSyncProfile] = useState<boolean>(false);
  const profileId = localStorage.getItem("profile_id");
  const [syncState, setSyncState] = useState<SyncState>(() => profileId ? "active" : "idle");
  const ws = useWebSocketConnection(isRecording || isSyncProfile)
  const [briefingList, setBriefingList] = useState<BriefingComponent[]>([])
  const [newFaceDetected, setNewFaceDetected] = useState<boolean>(false)

  SetWindowPosition()
  AutoResizeWindow()
  useBackendLifecycle()
  useBackendAudioProcess(ws, isRecording, profileId)
  VisitorFaceProcess(
    ws,
    isFaceCapture,
    profileId,
    (_profileId, _faceEmbedding) => { setNewFaceDetected(true) },
    setBriefingList
  )

  return (
    <main className="w-full p-2">
      {/* Glossy dark panel */}
      <div className="relative flex flex-col rounded-2xl overflow-hidden bg-zinc-900/90 backdrop-blur-2xl border border-white/6">
        {/* Gloss sheen overlay */}
        <div className="absolute inset-x-0 top-0 h-1/2 bg-linear-to-b from-white/5 to-transparent pointer-events-none z-0 rounded-t-2xl" />

        {/* Title bar */}
        <div className="relative z-10 flex items-center justify-between px-4 pt-1.5 pb-1 shrink-0">
          <span className="text-[11px] font-bold text-zinc-400 leading-none tracking-widest uppercase">Mosaic</span>
          <button
            onClick={() => getCurrentWindow().close().catch(console.error)}
            className="w-4 h-4 rounded-full flex items-center justify-center text-zinc-300 hover:text-zinc-100 transition-all duration-150 cursor-pointer"
            aria-label="Close"
          >
            <X size={14}/>
          </button>
        </div>
        {/* Divider */}
        <div className="relative z-10 h-px bg-zinc-800 shrink-0" />

        {/* Controls */}
        <div className="relative z-10 flex items-center justify-between px-4 pt-1.5 pb-2.5 shrink-0">
          <ProfileStatus syncState={syncState} onResync={() => { setSyncState("scanning"); setIsSyncProfile(true); }}/>
          <RecordButton
            ws={ws}
            syncState={syncState}
            setSyncState={setSyncState}
            isRecording={isRecording}
            setIsRecording={setIsRecording}
            setIsSyncCapture={setIsSyncProfile}
            setIsFaceCapture={setIsFaceCapture}
          />
        </div>

        {/* Divider */}
        <div className="relative z-10 h-px bg-zinc-800 shrink-0" />
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
