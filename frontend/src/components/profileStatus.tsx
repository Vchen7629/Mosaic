import { RefreshCcw } from 'lucide-react';

/**
 * Displays the current profile sync state in the header
 * @param syncState - controls what element to display
 * @param onResync - optional callback to re-trigger profile sync (shown when active)
 */
export const ProfileStatus = ({ syncState, onResync }: { syncState: string; onResync?: () => void }) => {
    if (syncState === "idle") {
        return (
          <span className="text-[12px] font-medium text-zinc-500 leading-none">No profile synced</span>
        )
    }

    if (syncState === "scanning") {
        return (
            <div className="flex items-center gap-2.5">
              <div className="relative w-5 h-5 flex items-center justify-center shrink-0">
                <div className="absolute inset-0 rounded-full border border-sky-400/20 animate-ping" />
                <div className="absolute inset-0.75 rounded-full border border-sky-400/40" />
                <div className="w-1.5 h-1.5 rounded-full bg-sky-400 shadow-[0_0_6px_rgba(56,189,248,0.8)]" />
              </div>
              <span className="text-[12px] font-medium text-zinc-400 leading-none">Looking for face…</span>
            </div>
        )
    }

    return (
      <div className="flex items-center gap-2">
        <div className="relative w-3 h-3 flex items-center justify-center shrink-0">
          <div className="absolute inset-0 rounded-full bg-emerald-400/20 animate-ping duration-300" />
          <div className="w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.7)]" />
        </div>
        <span className="text-[12px] font-semibold text-zinc-200 leading-none tracking-tight">Profile Synced</span>
        {onResync && (
          <button
            onClick={onResync}
            className="ml-0.5 flex items-center justify-center text-emerald-600 hover:text-emerald-400 transition-colors duration-150 cursor-pointer"
            aria-label="Re-sync profile"
          >
            <RefreshCcw size={13} />
          </button>
        )}
      </div>
    )
}
