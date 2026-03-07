import { StartAudioRecording, StopAudioRecording } from "../api/hooks/audio";

interface Props {
  isRecording: boolean;
  setIsRecording: (val: boolean) => void;
}

const StartRecordingButton = ({ isRecording, setIsRecording }: Props) => {
  const startMutation = StartAudioRecording(setIsRecording);
  const stopMutation = StopAudioRecording(setIsRecording);

  const handleToggle = () => {
    if (isRecording) {
      stopMutation.mutate();
    } else {
      startMutation.mutate();
    }
  };

  return (
    <button
      onClick={handleToggle}
      className={`flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-semibold transition-all duration-200 cursor-pointer border ${
        isRecording
          ? "bg-red-500/20 border-red-500/40 text-red-400 hover:bg-red-500/30"
          : "bg-emerald-500/20 border-emerald-500/40 text-emerald-400 hover:bg-emerald-500/30"
      }`}
    >
      <span
        className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${
          isRecording ? "bg-red-400 recording-dot" : "bg-emerald-400"
        }`}
      />
      {isRecording ? "Stop" : "Start"}
    </button>
  );
};

export default StartRecordingButton;
