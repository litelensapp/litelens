import { FC } from "react";
import type { TrayTab } from "./PodTray";

export const PodMetaStrip: FC<{ tab: TrayTab }> = ({ tab }) => (
  <>
    <span className="bg-muted/30 inline-flex items-center rounded-full border px-2 text-[11px]">
      {tab.ns}
    </span>
    <span className="text-muted-foreground/50 text-[11px]">·</span>
    {tab.ownerKind && tab.ownerName && (
      <>
        <span className="text-muted-foreground text-[11px]">
          {tab.ownerKind}: {tab.ownerName}
        </span>
        <span className="text-muted-foreground/50 text-[11px]">·</span>
      </>
    )}
    <span className="font-mono text-xs font-medium">Pod: {tab.pod}</span>
    <span className="bg-border h-4 w-px" />
  </>
);
