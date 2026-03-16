import { useEffect, useRef } from "react"

interface UseAudioCaptureProps {
    enabled: boolean
    onAudioData: (samples: Float32Array) => void
}

/**
 * @description Public hook for orchestrating audio capture loop
 * 
 * @param enabled 
 * @param onAudioData
 */
export function useAudioCapture({ enabled, onAudioData }: UseAudioCaptureProps) {
    const streamRef = useRef<MediaStream | null>(null)
    const audioContextRef = useRef<AudioContext | null>(null)
    const processorRef = useRef<ScriptProcessorNode | null>(null)
    
    function cleanup() {
        processorRef.current?.disconnect()
        audioContextRef.current?.close()
        streamRef.current?.getTracks().forEach(t => t.stop())
    }

    useEffect(() => {
        if (!enabled) return

        initAudioCapture(streamRef, audioContextRef, onAudioData, processorRef)
        return cleanup

    }, [enabled, onAudioData])
}

/**
 * @description Initializes the microphone audio stream
 * 
 * @param streamRef 
 * @param audioContextRef 
 * @param onAudioData 
 * @param processorRef 
 */
async function initAudioCapture(
    streamRef: React.RefObject<MediaStream | null>,
    audioContextRef: React.RefObject<AudioContext | null>,
    onAudioData: (samples: Float32Array) => void,
    processorRef: React.RefObject<ScriptProcessorNode | null>
) {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
        streamRef.current = stream

        const audioContext = new AudioContext({ sampleRate: 16000 })
        audioContextRef.current = audioContext

        const processor = setupAudioProcessor(stream, audioContext, onAudioData)
        processorRef.current = processor

    } catch (err) {
        console.error("Audio capture failed:", err)
    }
}

/**
 * @description Sets up Web Audio Api processor
 * 
 * @param stream 
 * @param audioContext 
 * @param onAudioData 
 * @returns 
 */
function setupAudioProcessor(
    stream: MediaStream,
    audioContext: AudioContext,
    onAudioData: (samples: Float32Array) => void
): ScriptProcessorNode {
    const source = audioContext.createMediaStreamSource(stream)
    const processor = audioContext.createScriptProcessor(4096, 1, 1)

    processor.onaudioprocess = (e) => {
        const samples = e.inputBuffer.getChannelData(0)
        console.log(`[Audio Capture] Captured ${samples.length} samples`)
        onAudioData(samples)
    }

    source.connect(processor)
    processor.connect(audioContext.destination)

    return processor
}

