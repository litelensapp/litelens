import { FC } from "react";
import { cn } from "@litelens/design-system";

export const SectionDivider: FC<{ label: string; className?: string }> = ({ label, className }) => (
  <div className={cn("bg-muted/40 text-muted-foreground text-h3 border-y px-4 py-2", className)}>
    {label}
  </div>
);
