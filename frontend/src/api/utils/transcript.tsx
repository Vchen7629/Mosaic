import { useMutation } from "@tanstack/react-query"

interface SaveConversationPayload {
    profile_id: string
    name: string
    timestamp: string
    conversation_summary: string
    topics: string[]
}
interface FetchBriefingPayload {

    profile_id: string
    name: string

}

export const SaveConversation = () => {
    return useMutation({
        mutationFn: async (payload: SaveConversationPayload) => {
            const res = await fetch("http://localhost:8000/conversation/save", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload),
            })
            if (!res.ok) throw new Error("Failed to save conversation")
        },
    })
}

export const FetchBriefing = () => {
    return useMutation({
        mutationFn: async (payload: FetchBriefingPayload) => {

            const res = await fetch("http://localhost:8000/conversation/fetch_briefing", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload),
            })

            if (!res.ok) throw new Error("Failed to fetch briefing")
            return res.json()

        }
    })


}