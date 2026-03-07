import { useMutation } from "@tanstack/react-query"

export const FetchTranscript = () => {
    return useMutation({
        mutationFn: async () => {
            
            const res = await fetch("http://localhost:8000/audio/start")

            if (!res.ok) throw new Error("Failed to start audio")

            return res.json()
            //conversation/summerise?q=(transcriptData)
        },
        

    })
}

//ignore this, leaving a note so I can find this easy
export const SummeriseGemeni = (transcript: string) => {

    return useMutation({
        mutationFn: async () => {
            
            const res = await fetch(`http://localhost:8000/conversation/summerise?q=${transcript}`)

            if (!res.ok) throw new Error("Failed to read and return transcript")

            return res.json()
        },
        

    })

}

