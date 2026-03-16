import { Camera, RefreshCcw } from 'lucide-react';

/**
 * Component that displays the currently logged in profile status for the current user
 * @param syncState 
 * @returns 
 */
export const ProfileStatus = ({ syncState }: { syncState: string }) => {
    return (
      <>
          {syncState === "idle" && (
              <div className="flex flex-col gap-px">
                <span className="text-[10px] font-medium text-zinc-600 leading-none tracking-wide uppercase">Patient</span>
                <span className="text-[12px] font-medium text-zinc-500 leading-none">No patient synced</span>
              </div>
          )}

          {syncState === "scanning" && (
            <div className="flex items-center gap-2.5">
              <div className="relative w-5 h-5 flex items-center justify-center shrink-0">
                <div className="absolute inset-0 rounded-full border border-sky-400/20 animate-ping" />
                <div className="absolute inset-0.75 rounded-full border border-sky-400/40" />
                <div className="w-1.5 h-1.5 rounded-full bg-sky-400 shadow-[0_0_6px_rgba(56,189,248,0.8)]" />
              </div>
              <div className="flex flex-col gap-px">
                <span className="text-[10px] font-medium text-zinc-500 leading-none tracking-wide uppercase">Scanning</span>
                <span className="text-[12px] font-medium text-zinc-400 leading-none">Looking for face…</span>
              </div>
            </div>
          )}

          {syncState === "active" && (
            <div className="flex flex-col gap-px">
              <span className="text-[10px] font-medium text-zinc-500 leading-none tracking-wide uppercase">Patient</span>
              <span className="text-[13px] font-semibold text-zinc-200 leading-none tracking-tight">Placehodler</span>
            </div>
          )}
        </>
    )
}

type Props = {
  syncState: string;
  setSyncState: any;
  setIsCapturingFace: (value: boolean) => void;
};

/**
 * The button to 
 * @param param0 
 * @returns 
 */
export const SyncProfileButton = ({ syncState, setSyncState, setIsCapturingFace }: Props) => {
  function handleSyncStart() {
    setSyncState("scanning");
    setIsCapturingFace(true);
  }

  function handleCancel() {
    setSyncState("idle");
    setIsCapturingFace(false);
  }

  function handleConfirm() {
    setSyncState("active");
    // TODO: persist patient to localStorage
  }



  return (
    <>
      {syncState === "idle" && (
        <button
          onClick={handleSyncStart}
          className="flex items-center gap-1.5 px-3 py-1 rounded-full text-[11px] font-semibold bg-zinc-800 border border-white/8 text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/80 hover:border-white/[0.14] transition-all duration-150 cursor-pointer"
        >
          <Camera size={15}/>
          Sync Patient
        </button>
      )}

      {syncState === "scanning" && (
        <button
          onClick={handleCancel}
          className="px-2.5 py-1 rounded-full text-[11px] font-semibold bg-zinc-800 border border-white/6 text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700 transition-all duration-150 cursor-pointer"
        >
          Cancel
        </button>
      )}

      {syncState === "active" && (
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5 px-2 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20">
            <div className="w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.7)] shrink-0" />
            <span className="text-[10px] font-semibold tracking-wide text-emerald-400">Active</span>
          </div>
          <button
            onClick={handleSyncStart}
            className="w-6 h-6 rounded-full flex items-center justify-center text-zinc-600 hover:text-zinc-300 hover:bg-white/8 transition-all duration-150 cursor-pointer"
            aria-label="Re-sync patient"
          >
            <RefreshCcw size={15}/>
          </button>
        </div>
      )}

    </>
  );
}