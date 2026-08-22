import { FC } from "react";
import { cn } from "@litelens/design-system";

export const SectionDivider: FC<{ label: string; className?: string }> = ({ label, className }) => (
  <div className={cn("text-h3 border-y bg-muted/40 px-4 py-2 text-muted-foreground", className)}>
    {label}
  </div>
);
