import "../App.css";

interface SumWindowProps {
  Name: string;
  Briefing: string;
}

const ConvoBriefingDisplay = ({ Name, Briefing }: SumWindowProps) => {
  return (
    <div className="pt-1.5 pb-2 w-full">
      <div
        className="relative flex flex-col rounded-2xl
          bg-zinc-900/90 backdrop-blur-2xl
          border border-white/6
          shadow-[inset_0_1px_0_rgba(255,255,255,0.1),0_8px_32px_rgba(0,0,0,0.6)]
          overflow-hidden"
      >
        {/* Gloss sheen */}
        <div className="absolute inset-x-0 top-0 h-1/3 bg-linear-to-b from-white/4 to-transparent pointer-events-none z-0" />

        {/* Violet top accent line */}
        <div className="absolute inset-x-0 top-0 h-px bg-linear-to-r from-transparent via-violet-500/40 to-transparent" />

        {/* Header */}
        <div className="relative z-10 flex items-center justify-between px-4 pt-3 pb-2">
          <div className="flex items-center gap-2">
            {/* Visitor avatar dot */}
            <div className="w-8 h-6 rounded-full bg-linear-to-br from-violet-600 to-indigo-600 flex items-center 
                    justify-center text-[12px] font-bold text-white shadow-[0_2px_6px_rgba(109,40,217,0.4)] shrink-0">
              {Name?.[0]?.toUpperCase()}
            </div>
            <span className="text-[16px] font-semibold text-zinc-100 tracking-tight">
              {Name}
            </span>
          </div>
          <span className="text-[15px] font-medium text-violet-400/70 tracking-widest uppercase">
            Briefing
          </span>
        </div>

        {/* Divider */}
        <div className="relative z-10 h-px bg-zinc-800 mx-4" />

        {/* Briefing text */}
        <p className="relative z-10 text-[15px] text-zinc-300 leading-relaxed px-4 pt-2.5 pb-3">
          {Briefing}
        </p>
      </div>
    </div>
  );
};

export default ConvoBriefingDisplay;
