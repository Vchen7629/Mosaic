import { useMutation } from "@tanstack/react-query"

const baseRoute = "http://localhost:8000/llm"

//ignore this, leaving a note so I can find this easy
export const SummarizeTranscript = () => {
    return useMutation({
        mutationFn: async (transcript: string) => {

            const res = await fetch(`${baseRoute}/summarize`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ conversation: transcript})
            })

            if (!res.ok) throw new Error("Failed to read and return transcript")

            return res.json()
        },

    })
}

interface BriefingPayload {
    patient_id: string
    name: string
}

export const GenerateBriefing = () => {
    return useMutation({
        mutationFn: async (payload: BriefingPayload) => {

            const res = await fetch(`${baseRoute}/generate_briefing`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload),
            })
            if (!res.ok) throw new Error("Failed to generate briefing")
        },

    })

}