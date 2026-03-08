import { useState, useEffect, useRef } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import "./App.css";
import StartRecordingButton from "./components/startRecordingBtn";
import { SetWindowPosition } from "./utils/setWindowPosition";
import { handleBackendLifecycle } from "./utils/handleBackendLifecycle";
import { useFaceDetection } from "./api/hooks/detection";
import NewFaceInput from "./components/newFaceInput";
import { AutoResizeWindow } from "./utils/autoResizeWindow";
import { FetchBriefing } from "./api/hooks/transcript";
import ConvoBriefingDisplay from "./components/convo_briefing";

function App() {
  const [isRecording, setIsRecording] = useState(false);
  const [newFaceName, setNewFaceName] = useState("");
  const patientId = localStorage.getItem("patient_id") ?? "test-patient";
  const { detectedName, unknownFaceDetected, confirmNewFace } = useFaceDetection(patientId, isRecording);
  const briefingRes = FetchBriefing();
  const briefingFetched = useRef(false);

  useEffect(() => {
    if (!isRecording) {
      briefingFetched.current = false;
      const timer = setTimeout(() => briefingRes.reset(), 700);
      return () => clearTimeout(timer);
    }
    if (detectedName && !briefingFetched.current) {
      briefingFetched.current = true;
      briefingRes.mutate({ patient_id: patientId, name: detectedName });
    }
  }, [detectedName, isRecording]);

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
          <span className="text-[15px] font-medium text-zinc-300 tracking-tight">
            Live Transcription
          </span>
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

      </div>

      {briefingRes.data && (
        <div className={`transition-[opacity,transform] duration-700 ease-out starting:opacity-0 starting:translate-y-3 ${isRecording ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'}`}>
          <ConvoBriefingDisplay Name={briefingRes.data.name} Briefing={briefingRes.data.summary}/>
        </div>
      )}

      
    </main>
  );
}



export default App;
