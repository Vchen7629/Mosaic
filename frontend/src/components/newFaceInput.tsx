import { useState } from "react"
import { useNewVisitorFaceRegister } from "../api/hooks/face"


type NewFaceInputProps = {
    ws: WebSocket | null
    faceEmbedding: string
    sessionToken: string
    onVisitorRegistered: (visitorId: string) => void
}

/**
 * Component that pops up when the application is recording the conversation and
 * a new visitor face shows up that wasnt registered in the database
 * allows users to enter a name and click save to register
 * @param ws -
 * @param faceEmbedding - 
 * @param sessionToken - 
 */
const NewFaceInput = ({ ws, faceEmbedding, sessionToken, onVisitorRegistered }: NewFaceInputProps) => {
    const [newFaceName, setNewFaceName] = useState<string>("")
    const [shouldRegister, setShouldRegister] = useState<boolean>(false)

    useNewVisitorFaceRegister(ws, shouldRegister, faceEmbedding, sessionToken, newFaceName, (visitorId) => {
        onVisitorRegistered(visitorId)
        setShouldRegister(false)
        setNewFaceName("")
    })

    function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
        if (e.key === "Enter" && newFaceName.trim()) {
            setShouldRegister(true)
        }
    }

    return (
        <div className="relative z-10 flex items-center gap-3 px-4 py-2.5 border-b border-zinc-800 bg-transparent">
            <span className="text-[11px] font-medium text-zinc-400 tracking-tight">New face</span>
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
                onClick={() => setShouldRegister(true)}
                disabled={!newFaceName.trim()}
                className="bg-emerald-500 text-zinc-950 hover:bg-emerald-400 active:bg-emerald-600 transition-colors 
                            disabled:bg-emerald-600 disabled:text-zinc-900 disabled:cursor-not-allowed
                            shrink-0 text-[11px] font-medium px-3 py-1 rounded-md "
            >
                Save
            </button>
        </div>
    )
}

export default NewFaceInput