import { useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import "./App.css";
import StartRecordingButton from "./components/startRecordingBtn";
import { SetWindowPosition } from "./utils/setWindowPosition";
import { handleBackendLifecycle } from "./utils/handleBackendLifecycle";
import { useFaceDetection } from "./api/hooks/detection";
import NewFaceInput from "./components/newFaceInput";
import { AutoResizeWindow } from "./utils/autoResizeWindow";

interface Entry {
  id: number;
  name: string;
  text: string;
  timestamp: string;
}

const MOCK_ENTRIES: Entry[] = [
  { id: 1, name: "Alice", text: "Can everyone hear me okay?", timestamp: "0:04" },
  { id: 2, name: "Bob", text: "Yeah, audio sounds good on my end.", timestamp: "0:12" },
  { id: 3, name: "Carol", text: "Let's go over the agenda for today.", timestamp: "0:21" },
  { id: 4, name: "Alice", text: "Sure, I'll share my screen in a moment.", timestamp: "0:35" },
];

function App() {
  const [isRecording, setIsRecording] = useState(false);
  const [entries] = useState<Entry[]>(MOCK_ENTRIES);
  const [newFaceName, setNewFaceName] = useState("");
  const patientId = localStorage.getItem("patient_id") ?? "test-patient";
  const { detectedName, unknownFaceDetected, confirmNewFace } = useFaceDetection(patientId, isRecording);

  SetWindowPosition()
  AutoResizeWindow()
  handleBackendLifecycle()

  return (
    <main className="w-full p-2">
      {/* Glossy dark panel */}
      <div className="relative flex flex-col rounded-2xl overflow-hidden
                      bg-zinc-900/90 backdrop-blur-2xl
                      border border-white/[0.06]
                      shadow-[inset_0_1px_0_rgba(255,255,255,0.1),0_8px_32px_rgba(0,0,0,0.6)]">

        {/* Gloss sheen overlay */}
        <div className="absolute inset-x-0 top-0 h-1/2 bg-gradient-to-b from-white/[0.05] to-transparent pointer-events-none z-0 rounded-t-2xl" />

        {/* Header */}
        <div className="relative z-10 flex items-center justify-between px-4 py-3 flex-shrink-0">
          <div className="flex items-center gap-2.5">
            <span
              className={`w-2 h-2 rounded-full flex-shrink-0 transition-colors duration-300 ${
                isRecording
                  ? "bg-red-500 recording-dot shadow-[0_0_8px_rgba(239,68,68,0.7)]"
                  : "bg-zinc-600"
              }`}
            />
            <span className="text-[13px] font-medium text-zinc-300 tracking-tight">
              Live Transcription
            </span>
          </div>
          <div className="flex items-center gap-2">
            <StartRecordingButton
              isRecording={isRecording}
              setIsRecording={setIsRecording}
              detectedName={detectedName}
              patientId={patientId}
            />
            <button
              onClick={() => getCurrentWindow().close().catch(console.error)}
              className="w-6 h-6 rounded-full flex items-center justify-center text-zinc-500 hover:text-zinc-200 hover:bg-white/[0.08] transition-all duration-150 cursor-pointer"
              aria-label="Close"
            >
              <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                <path d="M1 1l8 8M9 1L1 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
              </svg>
            </button>
          </div>
        </div>

        {/* Divider */}
        <div className="relative z-10 h-px bg-zinc-800 flex-shrink-0" />

        {unknownFaceDetected && (
          <NewFaceInput newFaceName={newFaceName} setNewFaceName={setNewFaceName} confirmNewFace={confirmNewFace}/>
        )}

        {/* Entries list */}
        <div className="relative z-10 flex-1 overflow-y-auto">
          {entries.length === 0 ? (
            <p className="text-center text-zinc-600 text-xs py-10">No transcriptions yet.</p>
          ) : (
            entries.map((entry) => (
              <div
                key={entry.id}
                className="flex items-start gap-3 px-4 py-3 hover:bg-white/[0.04] transition-colors duration-100 border-b border-zinc-800/60 last:border-b-0"
              >
                
                <div className="w-7 h-7 rounded-full bg-gradient-to-br from-violet-600 to-indigo-600 flex items-center justify-center text-[11px] font-semibold text-white flex-shrink-0 mt-0.5 shadow-[0_2px_6px_rgba(109,40,217,0.4)]">
                  {entry.name[0]}
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-baseline gap-2 mb-1">
                    <span className="text-[13px] font-semibold text-zinc-100">{entry.name}</span>
                    <span className="text-[11px] text-zinc-600">{entry.timestamp}</span>
                  </div>
                  <p className="text-[12px] text-zinc-400 leading-snug truncate">{entry.text}</p>
                </div>
              </div>
            ))
          )}
        </div>
        
      </div>
    </main>
  );
}

export default App;
