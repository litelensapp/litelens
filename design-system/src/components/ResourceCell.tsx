import { FC } from "react";
import { cn } from "../utils/common";

function barColor(percent: number): string {
  if (percent <= 50) return "bg-success";
  if (percent <= 80) return "bg-warning";
  return "bg-destructive";
}

interface ResourceCellProps {
  label: string;
  percent: number;
}

export const ResourceCell: FC<ResourceCellProps> = ({ label, percent }) => (
  <div className="flex min-w-36 flex-col gap-1">
    <div className="bg-muted h-1.5 w-full rounded-full">
      {percent > 0 && (
        <div
          className={cn("transition-interactive h-full rounded-full", barColor(percent))}
          style={{ width: `${Math.min(percent, 100)}%` }}
        />
      )}
    </div>
    <span className="text-muted-foreground text-xs">{label}</span>
  </div>
);
