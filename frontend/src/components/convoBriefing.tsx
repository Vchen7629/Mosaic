import "../App.css";

interface ConvoBriefingProps {
  VisitorName: string;
  Briefing: string;
}

/**
 * Component for a single visitor briefing fetched from the database for a visitor
 * @param VisitorName - the name of the visitor the briefing belongs to
 * @param Briefing - the 3-4 sentence briefing text
 */
const ConvoBriefingDisplay = ({ VisitorName, Briefing }: ConvoBriefingProps) => {
  return (
    <div className="w-full py-2">
      <div className="briefing-card relative flex flex-col rounded-lg px-3 py-2.5 overflow-hidden">
        {/* Header with name and label */}
        <div className="relative z-10 flex items-center justify-between gap-2 mb-1.5">
          <div className="flex items-center gap-2 min-w-0">
            <div className="w-6 h-6 rounded-full bg-linear-to-br from-teal-700/50 to-teal-800/50 flex items-center justify-center text-[11px] font-semibold text-teal-100 shrink-0">
              {VisitorName?.[0]?.toUpperCase()}
            </div>
            <span className="text-[13px] font-medium text-zinc-200 truncate">
              {VisitorName ? VisitorName : "Unknown Visitor"}
            </span>
          </div>
          <span className="text-[11px] font-medium text-emerald-400/70 uppercase shrink-0">
            Briefing
          </span>
        </div>

        {/* Briefing text */}
        <p className="relative z-10 text-[13px] text-zinc-400 leading-relaxed">
          {Briefing ? Briefing : <span className="px-2">
            No briefing generated yet, stop the recording to generate a briefing based on the current conversation
          </span>}
        </p>
      </div>
    </div>
  );
};

export default ConvoBriefingDisplay;
