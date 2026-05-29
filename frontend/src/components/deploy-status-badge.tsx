import type { RiskLevel, DeployStatus } from "@/lib/types";
import { cn } from "@/lib/utils";

const riskColors: Record<RiskLevel, string> = {
  low:      "bg-emerald-950 text-emerald-400 border-emerald-800",
  medium:   "bg-amber-950 text-amber-400 border-amber-800",
  high:     "bg-orange-950 text-orange-400 border-orange-800",
  critical: "bg-red-950 text-red-400 border-red-800",
};

const statusColors: Record<DeployStatus, string> = {
  pending:  "bg-zinc-900 text-zinc-400 border-zinc-700",
  approved: "bg-emerald-950 text-emerald-400 border-emerald-800",
  rejected: "bg-red-950 text-red-400 border-red-800",
  skipped:  "bg-zinc-900 text-zinc-500 border-zinc-700",
  voided:   "bg-zinc-900 text-zinc-600 border-zinc-800",
};

export function RiskBadge({ level }: { level: RiskLevel }) {
  return (
    <span className={cn("inline-flex items-center rounded border px-2 py-0.5 text-xs font-mono uppercase tracking-widest", riskColors[level])}>
      {level}
    </span>
  );
}

export function StatusBadge({ status }: { status: DeployStatus }) {
  return (
    <span className={cn("inline-flex items-center rounded border px-2 py-0.5 text-xs font-mono uppercase tracking-widest", statusColors[status])}>
      {status}
    </span>
  );
}
