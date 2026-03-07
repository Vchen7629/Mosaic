
const NewFaceInput = ({ newFaceName, setNewFaceName, confirmNewFace }: any) => {
    function handleClick() {
        if (newFaceName.trim()) {
            confirmNewFace(newFaceName.trim())
            setNewFaceName("")
        }
    }

    function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
        if (e.key === "Enter") handleClick()
    }

    return (
        <div className="relative z-10 flex items-center gap-3 px-4 py-2.5 border-b border-zinc-800 bg-transparent">
            <div className="flex items-center gap-1.5 flex-shrink-0">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.6)]" />
                <span className="text-[11px] font-medium text-zinc-400 tracking-tight">New face</span>
            </div>
            <input
                type="text"
                value={newFaceName}
                onChange={(e) => setNewFaceName(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Enter name…"
                autoFocus
                className="flex-1 bg-zinc-800 rounded-md px-2.5 py-1 text-[12px] text-zinc-100 placeholder-zinc-600 outline-none ring-1 ring-zinc-700 focus:ring-emerald-500/50 transition-all"
            />
            <button
                onClick={handleClick}
                className="flex-shrink-0 text-[11px] font-medium px-3 py-1 rounded-md bg-emerald-500 text-zinc-950 hover:bg-emerald-400 active:bg-emerald-600 transition-colors"
            >
                Save
            </button>
        </div>
    )
}

export default NewFaceInput