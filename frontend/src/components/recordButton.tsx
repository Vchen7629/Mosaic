import { Camera } from "lucide-react";
import { SyncProfileProcess } from "../api/utils/face";
import { SyncState } from "../types/profle";
import { Dispatch, SetStateAction } from "react";
import { BriefingComponent } from "../types/briefing";

type RecordButtonProps = {
  ws: WebSocket | null;
  syncState: SyncState;
  setSyncState: Dispatch<SetStateAction<SyncState>>;
  isRecording: boolean;
  profileId: string;
  visitorIds: string[];
  setIsRecording: (val: boolean) => void;
  setIsSyncCapture: (val: boolean) => void;
  setIsFaceCapture: (val: boolean) => void;
  setVisitorIds: Dispatch<SetStateAction<string[]>>
  setBriefingList: Dispatch<SetStateAction<BriefingComponent[]>>
};

/**
 * Unified button that handles the full flow: sync profile → start/stop recording.
 * Has possible states states: idle → scanning → active/start → active/stop
 * @param ws
 */
export const RecordButton = ({ ws,  syncState, setSyncState, isRecording, profileId, visitorIds, setIsRecording, setIsSyncCapture, setIsFaceCapture, setVisitorIds, setBriefingList }: RecordButtonProps) => {
  SyncProfileProcess(ws, syncState === "scanning", (_profileId) => {
    setSyncState("active");
    setIsSyncCapture(false);
  });

  function handleSyncStart() {
    setSyncState("scanning");
    setIsSyncCapture(true);
  }

  function handleCancel() {
    setSyncState("idle");
    setIsSyncCapture(false);
  }

  function handleRecordToggle() {
    setIsRecording(!isRecording);
    setIsFaceCapture(!isRecording);
  }

  function handleStopRecording() {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: "save_audio_transcript",
        profile_id: profileId,
        visitor_id: visitorIds[0] ?? "0",
      }))
    }
    setIsRecording(false)
    setIsFaceCapture(false)
    setVisitorIds([])
    setBriefingList([])
  }

  if (syncState === "idle") {
    return (
      <button
        onClick={handleSyncStart}
        className="flex items-center gap-1.5 px-3 py-1 rounded-full text-[11px] font-semibold bg-zinc-800 border border-white/8 text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/80 hover:border-white/[0.14] transition-all duration-150 cursor-pointer"
      >
        <Camera size={15} />
        Sync Profile
      </button>
    );
  }

  if (syncState === "scanning") {
    return (
      <button
        onClick={handleCancel}
        className="px-2.5 py-1 rounded-full text-[11px] font-semibold bg-zinc-800 border border-white/6 text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700 transition-all duration-150 cursor-pointer"
      >
        Cancel
      </button>
    );
  }

  // active state
  return (
    <div className="flex items-center gap-1.5">
      <button
        onClick={isRecording ? handleStopRecording : handleRecordToggle}
        className={`flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-semibold transition-all duration-200 border ${
          isRecording
            ? "bg-red-500/20 border-red-500/40 text-red-400 hover:bg-red-500/30 cursor-pointer"
            : "bg-emerald-500/20 border-emerald-500/40 text-emerald-400 hover:bg-emerald-500/30 cursor-pointer"
        }`}
      >
        <span
          className={`w-1.5 h-1.5 rounded-full shrink-0 ${
            isRecording ? "bg-red-400 recording-dot" : "bg-emerald-400"
          }`}
        />
        {isRecording ? "Stop" : "Start"}
      </button>
    </div>
  );
};

