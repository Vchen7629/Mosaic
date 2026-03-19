import { Camera } from "lucide-react";
import { SyncState } from "../types/profle";
import { useSyncProfileProcess } from "../api/hooks/face";

type RecordButtonProps = {
  ws: WebSocket | null;
  syncState: SyncState;
  isRecording: boolean;
  profileId: string;
  visitorIds: string[];
  onSyncStart: () => void;
  onSyncCancel: () => void;
  onSyncComplete: () => void;
  onRecordingStart: () => void;
  onRecordingStop: () => void
};

/**
 * Public exposed button that displays the different buttons depending on state
 * @param ws - the websocket connection
 * @param syncState -
 * @param isRecording -
 * @param profileID - currently synced profile id
 * @param visitorIds - array of visitorIds strings to save transcripts with
 * @param onSyncStart -
 * @param onSyncCancel - 
 * @param onSyncComplete -
 * @param onRecordingStart -
 * @param onRecordingStop
 */
export const RecordButton = ({ 
  ws, syncState, isRecording, profileId, visitorIds, 
  onSyncStart, onSyncCancel, onSyncComplete, onRecordingStart, onRecordingStop
}: RecordButtonProps) => {
  useSyncProfileProcess(ws, syncState === "scanning", (_profileId) => { onSyncComplete() });

  function handleStopRecording() {
    if (ws?.readyState === WebSocket.OPEN && visitorIds.length > 0) {
      ws.send(JSON.stringify({
        type: "save_audio_transcript",
        profile_id: profileId,
        visitor_id_list: visitorIds,
      }))
    }
    onRecordingStop()
  }

  if (syncState === "idle") return <SyncIdleButton onSyncStart={onSyncStart}/>
  if (syncState === "scanning") return <SyncCancelButton onSyncCancel={onSyncCancel}/>
  return <ActiveRecordButton isRecording={isRecording} onStart={onRecordingStart} onStop={handleStopRecording}/>
};

/**
 * Button component that appears when the user hasn't synced their profile yet
 * @param onSyncStart - passed in function that updates state
 */
const SyncIdleButton = ({ onSyncStart }: { onSyncStart: () => void }) => {
  return (
      <button
        onClick={onSyncStart}
        className="flex items-center gap-1.5 px-3 py-1 rounded-full text-[11px] font-semibold bg-zinc-800 border border-white/8 text-zinc-400
           hover:text-zinc-200 hover:bg-zinc-700/80 hover:border-white/[0.14] transition-all duration-150 cursor-pointer"
      >
        <Camera size={15} />
        Sync Profile
      </button>
  );
}

/**
 * Button component that appears when its syncing and the user wants to cancel
 * @param onSyncCancel - passed in function that updates state
 */
const SyncCancelButton = ({ onSyncCancel }: { onSyncCancel: () => void }) => {
  return (
      <button
        onClick={onSyncCancel}
        className="px-2.5 py-1 rounded-full text-[11px] font-semibold bg-zinc-800 border border-white/6 text-zinc-400
           hover:text-zinc-200 hover:bg-zinc-700 transition-all duration-150 cursor-pointer"
      >
        Cancel
      </button>
  );
}

/**
 * Button that controls starting and stopping recording conversation. Only appears if the profile is synced
 * @param isRecording - boolean representing whether the application is recording or not
 * @param onStart - passed in function that updates state to make the recording start
 * @param onStop - passed in function that updates state to make recording stop and save to db
 */
const ActiveRecordButton = (
  { isRecording, onStart, onStop }: { isRecording: boolean; onStart: () => void; onStop: () => void}
) => {
  return (
    <div className="flex items-center gap-1.5">
      <button
        onClick={isRecording ? onStop : onStart}
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
}