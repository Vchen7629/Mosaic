import { useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { SetWindowPosition } from "./utils/setWindowPosition";
import { useBackendLifecycle } from "./utils/handleBackendLifecycle";
import { AutoResizeWindow } from "./utils/autoResizeWindow";
import ConvoBriefingDisplay from "./components/convoBriefing";
import { useBackendAudioProcess } from "./api/hooks/audio";
import { VisitorFaceProcess } from "./api/utils/face";
import { useWebSocketConnection } from "./api/hooks/useWebSocket";
import { ProfileStatus } from "./components/profileStatus";
import { BriefingComponent } from "./types/briefing";
import { SyncState } from "./types/profle";
import { RecordButton } from "./components/recordButton";
import { X } from "lucide-react";
import NewFaceInput from "./components/newFaceInput";
import "./App.css";

function App() {
  const [isRecording, setIsRecording] = useState(false);
  const [isFaceCapture, setIsFaceCapture] = useState<boolean>(false);
  const [isSyncProfile, setIsSyncProfile] = useState<boolean>(false);
  const profileId = localStorage.getItem("profile_id");
  const [syncState, setSyncState] = useState<SyncState>(() => profileId ? "active" : "idle");
  /*const decoder = false
  const ws = useWebSocketConnection(decoder || isSyncProfile)*/
  const ws = useWebSocketConnection(isRecording || isSyncProfile)
  const [briefingList1, setBriefingList] = useState<BriefingComponent[]>([
    {
      "visitorName": "Sam", 
      "briefingText": "Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum."
    },
    {"visitorName": "Max", "briefingText": "lorem ipsum"}
  ])
  const [briefingList2, setBriefingList2] = useState<BriefingComponent[]>([])
  const [newFaceDetected, setNewFaceDetected] = useState<boolean>(false)
  const [newFaceName, setNewFaceName] = useState<string>("")

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
          setSyncState={setSyncState}
          isRecording={isRecording}
          setIsRecording={setIsRecording}
          setIsSyncCapture={setIsSyncProfile}
          setIsFaceCapture={setIsFaceCapture}
        />
      </section>
      <hr className={`border-zinc-700 ${isRecording ? 'opacity-100' : 'opacity-0'}`}/>
      <section className={`expand-wrap ${isRecording ? 'expand-open' : 'expand-closed'}`}>
        <div className="expand-inner">
          <div className="w-full h-fit min-h-20 max-h-120">
            <div className={`expand-wrap ${newFaceDetected ? 'expand-open' : 'expand-closed'}`}>
              <div className="expand-inner">
                <NewFaceInput newFaceName={newFaceName} setNewFaceName={setNewFaceName} confirmNewFace={confirm}/>
              </div>
            </div>
            {briefingList1.length > 0 ? (
              briefingList1.map((briefing: BriefingComponent, i) => (
                <div key={i}>
                  <ConvoBriefingDisplay VisitorName={briefing.visitorName} Briefing={briefing.briefingText}/>
                </div>
              ))
            ) : (
              <div className="flex rounded-lg px-3 py-2.5 overflow-hidden text-center items-center justify-center">
                <span className="mt-4 text-sm text-zinc-400">No Briefings</span>
              </div>
            )}
          </div>
        </div>
      </section>
    </main>
  );
}



export default App;
