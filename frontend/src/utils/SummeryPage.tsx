import "../App.css";
import {FetchBriefing} from "../api/hooks/transcript"

interface SumWindowProps {
  Name: string;
  Sum: any;
  Used: boolean;
}

const SumWindow = ({ Name, Sum, Used }: SumWindowProps) => {
  if (Used) {
    return (
      <main className="w-full p-2">
        {/* Second Dark Panel */}
        <div
          className="relative flex flex-col rounded-2xl overflow-y-auto
          bg-zinc-900/90 backdrop-blur-2xl
          border border-white/[0.06]
          shadow-[inset_0_1px_0_rgba(255,255,255,0.1),0_8px_32px_rgba(0,0,0,0.6)]"
        >
          <span className="text-[12px] font-semibold text-zinc-100 overflow-y-auto ml-3">
            {Name}
          </span>

          <p className="text-[14px] text-zinc-300 leading-snug truncate overflow-y-auto ml-3">
            {Sum}
          </p>
        </div>
      </main>
    );
  }

  return null;
};

export default SumWindow;