import { StartAudioRecording, StopAudioRecording } from "../api/hooks/audio";
import { GenerateBriefing, SummarizeTranscript } from "../api/hooks/llm";
import { SaveConversation } from "../api/hooks/transcript";


interface Props {
  isRecording: boolean;
  setIsRecording: (val: boolean) => void;
  detectedName: string | null;
  patientId: string;
}

const StartRecordingButton = ({ isRecording, setIsRecording, detectedName, patientId }: Props) => {
  const startMutation = StartAudioRecording(setIsRecording);
  const stopMutation = StopAudioRecording(setIsRecording);
  const summarizeMutation = SummarizeTranscript();
  const saveMutation = SaveConversation();
  const briefingMutation = GenerateBriefing();

  async function handleStop() {
    try {
      const { transcript } = await stopMutation.mutateAsync(undefined);
      if (!transcript || !detectedName) return;

      const summary = await summarizeMutation.mutateAsync(transcript);

      await saveMutation.mutateAsync({
        patient_id: patientId,
        name: detectedName,
        timestamp: new Date().toISOString(),
        conversation_summary: summary,
        topics: [],
      });

      await briefingMutation.mutateAsync({ patient_id: patientId, name: detectedName });
    } catch (e) {
      console.error("handleStop failed:", e);
    }
  };

  function handleToggle() {
    if (isRecording) {
      handleStop();
    } else {
      startMutation.mutate();
    }
  };

  const isStartPending = startMutation.isPending;
  const isStopPending = stopMutation.isPending;

  const label = isStartPending
    ? "Starting..."
    : isStopPending
    ? isRecording
      ? "Stopping..."
      : "Transcribing..."
    : isRecording
    ? "Stop"
    : "Start";

  return (
    <button
      onClick={handleToggle}
      disabled={isStartPending || isStopPending}
      className={`flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-semibold transition-all duration-200 border ${
        isStartPending || isStopPending
          ? "bg-red-700/20 border-red-700/40 text-red-500 hover:bg-red-500/30 cursor-not-allowed"
          : isRecording
          ? "bg-red-500/20 border-red-500/40 text-red-400 hover:bg-red-500/30 cursor-pointer"
          : "bg-emerald-500/20 border-emerald-500/40 text-emerald-400 hover:bg-emerald-500/30 cursor-pointer"
      }`}
    >
      <span
        className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${
          isRecording ? "bg-red-400 recording-dot" : "bg-emerald-400"
        }`}
      />
      {label}
    </button>
  );
};

export default StartRecordingButton;
