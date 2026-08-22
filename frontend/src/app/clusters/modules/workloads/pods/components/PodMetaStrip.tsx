import { FC } from "react";
import type { TrayTab } from "./PodTray";

export const PodMetaStrip: FC<{ tab: TrayTab }> = ({ tab }) => (
  <>
    <span className="inline-flex items-center rounded-full border bg-muted/30 px-2 text-[11px]">
      {tab.ns}
    </span>
    <span className="text-[11px] text-muted-foreground/50">·</span>
    {tab.ownerKind && tab.ownerName && (
      <>
        <span className="text-[11px] text-muted-foreground">
          {tab.ownerKind}: {tab.ownerName}
        </span>
        <span className="text-[11px] text-muted-foreground/50">·</span>
      </>
    )}
    <span className="font-mono text-xs font-medium">Pod: {tab.pod}</span>
    <span className="h-4 w-px bg-border" />
  </>
);
