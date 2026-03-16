
interface Props {
  profileID: string | null;
  isRecording: boolean;
  setIsRecording: (val: boolean) => void;
  isCapturingFace: boolean;
  setIsCapturingFace: (val: boolean) => void;
}

/**
 * @description - Button component that starts the recording and sets
 * is Recording state to correct one to trigger
 * @param profileID - passed in so we can disable recording if the profile isnt synced
 * @param isRecording - boolean 
 * @param setIsRecording
 * @returns 
 */
const StartRecordingButton = ({ profileID, isRecording, setIsRecording, isCapturingFace, setIsCapturingFace }: Props) => {
  const label = isRecording ? "Stop" : "Start";

  function handleToggle() {
    setIsRecording(!isRecording)
    setIsCapturingFace(!isCapturingFace)
    console.log(profileID)
  }

  return (
    <button
      disabled={!profileID}
      onClick={() => handleToggle()}
      className={`flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-semibold transition-all duration-200 border ${
        isRecording
          ? "bg-red-500/20 border-red-500/40 text-red-400 hover:bg-red-500/30 cursor-pointer"
          : "bg-emerald-500/20 border-emerald-500/40 text-emerald-400 hover:bg-emerald-500/30 cursor-pointer disabled:bg-emerald-600/20 disabled:cursor-not-allowed"
      }`}
    >
      <span
        className={`w-1.5 h-1.5 rounded-full shrink-0 ${
          isRecording ? "bg-red-400 recording-dot" : "bg-emerald-400"
        }`}
      />
      {label}
    </button>
  );
};

export default StartRecordingButton;
