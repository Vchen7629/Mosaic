import { useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { useBackendLifecycle } from "./hooks/useBackendLifecycle";
import { useAutoResizeWindow } from "./hooks/useAutoResizeWindow";
import ConvoBriefingDisplay from "./components/convoBriefing";
import { useBackendAudioProcess } from "./api/hooks/audio";
import { useWebSocketConnection } from "./api/hooks/useWebSocket";
import { ProfileStatus } from "./components/profileStatus";
import { BriefingComponent } from "./types/briefing";
import { SyncState } from "./types/profle";
import { RecordButton } from "./components/recordButton";
import { LoaderCircle, X } from "lucide-react";
import NewFaceInput from "./components/newFaceInput";
import "./App.css";
import { useVisitorFace } from "./api/hooks/face";
import { useSetWindowPosition } from "./hooks/useSetWindowPosition";

function App() {
  const [isRecording, setIsRecording] = useState(false);
  const [isFaceCapture, setIsFaceCapture] = useState<boolean>(false);
  const [isSyncProfile, setIsSyncProfile] = useState<boolean>(false);
  const profileId = localStorage.getItem("profile_id");
  const [syncState, setSyncState] = useState<SyncState>(() => profileId ? "active" : "idle");
  /*const decoder = false
  const ws = useWebSocketConnection(decoder || isSyncProfile)*/
  const ws = useWebSocketConnection(isRecording || isSyncProfile)
  const [briefingList1, setBriefingList] = useState<BriefingComponent[]>([])
  const [newFaceDetected, setNewFaceDetected] = useState<boolean>(false)
  const [pendingFaceEmbedding, setPendingFaceEmbedding] = useState<string>("")
  const [visitorIds, setVisitorIds] = useState<string[]>([])

  useSetWindowPosition()
  useAutoResizeWindow()
  useBackendLifecycle()
  useBackendAudioProcess(ws, isRecording, profileId)
  useVisitorFace(
    ws,
    isFaceCapture,
    profileId,
    (_profileId, faceEmbedding) => { setNewFaceDetected(true); setPendingFaceEmbedding(faceEmbedding) },
    (visitorId) => { setVisitorIds(prev => [...prev, visitorId]) },
    (briefing) => setBriefingList(prev => [...prev, briefing])
  )

  return (
    <main className="w-full p-2 relative flex flex-col rounded-2xl overflow-hidden bg-zinc-900/90 backdrop-blur-2xl border border-white/6">
      {/* Gloss sheen overlay */}
      <div className="absolute inset-x-0 top-0 h-1/2 bg-linear-to-b from-white/5 to-transparent pointer-events-none z-0 rounded-t-2xl" />
      <header className="flex items-center justify-between px-4 pt-1.5 pb-1">
        <span className="text-[11px] font-bold text-zinc-400 leading-none uppercase">Mosaic</span>
        <button
          onClick={() => getCurrentWindow().close().catch(console.error)}
          className="w-4 h-4 rounded-full flex items-center justify-center text-zinc-300 hover:text-zinc-100 transition-all duration-150 cursor-pointer"
          aria-label="Close"
        >
          <X size={14}/>
        </button>
      </header>
      <section className="flex items-center justify-between px-4 pt-1.5 pb-2.5">
        <ProfileStatus syncState={syncState} onResync={() => { setSyncState("scanning"); setIsSyncProfile(true); }}/>
        <RecordButton
          ws={ws}
          syncState={syncState}
          isRecording={isRecording}
          profileId={profileId ?? ""}
          visitorIds={visitorIds}
          onSyncStart={() => { setSyncState("scanning"); setIsSyncProfile(true); }}
          onSyncCancel={() => { setSyncState("idle"); setIsSyncProfile(false); }}
          onSyncComplete={() => { setSyncState("active"); setIsSyncProfile(false); }}
          onRecordingStart={() => { setIsRecording(true); setIsFaceCapture(true); }}
          onRecordingStop={() => { setIsRecording(false); setIsFaceCapture(false); setVisitorIds([]); setBriefingList([]); }}
        />
      </section>
      <hr className={`border-zinc-700 ${isRecording ? 'opacity-100' : 'opacity-0'}`}/>
      <section className={`expand-wrap ${isRecording ? 'expand-open' : 'expand-closed'}`}>
        <div className="expand-inner">
          <div className="w-full h-fit min-h-20 max-h-120">
            <div className={`expand-wrap ${newFaceDetected ? 'expand-open' : 'expand-closed'}`}>
              <div className="expand-inner">
                <NewFaceInput 
                  ws={ws} 
                  faceEmbedding={pendingFaceEmbedding} 
                  profileId={profileId ?? ""}
                  onVisitorRegistered={(visitorId => {
                    setVisitorIds(prev => [...prev, visitorId])
                    setNewFaceDetected(false)
                  })}
                />
              </div>
            </div>
            {briefingList1.length > 0 ? (
              briefingList1.map((briefing: BriefingComponent, i) => (
                <div key={i}>
                  <ConvoBriefingDisplay VisitorName={briefing.visitorName} Briefing={briefing.briefingText}/>
                </div>
              ))
            ) : (
              <div className="flex rounded-lg h-20 px-3 py-2.5 space-x-2 overflow-hidden text-center items-center justify-center">
                <LoaderCircle className="animate-spin text-emerald-500"/>
                <span className="text-sm text-zinc-400">Loading Briefings...</span>
              </div>
            )}
          </div>
        </div>
      </section>
    </main>
  );
}



export default App;
