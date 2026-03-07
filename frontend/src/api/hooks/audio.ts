import { useMutation } from "@tanstack/react-query"


export const StartAudioRecording = (setIsRecording: (val: boolean) => void) => {
    return useMutation({
        mutationFn: async () => {
            const res = await fetch("http://localhost:8000/audio/start")

            if (!res.ok) throw new Error("Failed to start audio")

            return res.json()
        },
        onSuccess: () => setIsRecording(true),
        onError: () => setIsRecording(false),
    })
}

export const StopAudioRecording = (setIsRecording: (val: boolean) => void) => {
    return useMutation({
        mutationFn: async () => {
            const res = await fetch("http://localhost:8000/audio/stop")

            if (!res.ok) throw new Error("Failed to stop audio")

            return res.json()
        },
        onSuccess: () => setIsRecording(false),
        onError: () => setIsRecording(true),
    })
}
