import { Button, LineIcon, Loader2Icon, cn } from "@litelens/design-system";
import { FC, useEffect, useRef } from "react";
import { useConnectStatusEvents } from "./shared/hooks/async-events/useConnectStatusEvents";

interface ConnectingViewProps {
  contextName: string;
  failed: boolean;
  onReconnect?: () => void;
  onOpenClusterSettings?: () => void;
}

const PALETTE = [
  "#2563eb",
  "#7c3aed",
  "#15803d",
  "#ea580c",
  "#db2777",
  "#0e7490",
  "#d97706",
  "#b91c1c",
  "#4f46e5",
  "#0f766e",
];

function clusterColor(name: string): string {
  const hash = Array.from(name).reduce((acc, c) => acc + (c.codePointAt(0) ?? 0), 0);
  return PALETTE[hash % PALETTE.length];
}

function clusterInitials(name: string): string {
  const parts = name.split(/[-_\s]+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}

export const ConnectingView: FC<ConnectingViewProps> = ({
  contextName,
  failed,
  onReconnect,
  onOpenClusterSettings,
}) => {
  const lines = useConnectStatusEvents(contextName);

  const bottomRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines]);

  const statusLabel = failed ? "Connection failed" : "Connecting...";

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-6 p-8">
      <div className="flex flex-col items-center gap-3">
        <div
          className="flex h-16 w-16 items-center justify-center rounded-2xl text-lg font-bold text-white shadow-lg"
          style={{ backgroundColor: clusterColor(contextName) }}
        >
          {clusterInitials(contextName)}
        </div>
        <div className="text-center">
          <p className="text-foreground font-semibold">{contextName}</p>
          <p className="text-muted-foreground text-sm">{statusLabel}</p>
        </div>
      </div>

      <div className="bg-muted/30 border-border w-full max-w-lg rounded-lg border p-4 font-mono text-sm">
        {lines.length === 0 && (
          <div className="text-muted-foreground flex items-center gap-2">
            <Loader2Icon className="h-3.5 w-3.5 animate-spin" />
            <span>Initializing...</span>
          </div>
        )}
        {lines.map((line, i) => {
          const isLast = i === lines.length - 1;
          const isSpinning = isLast && !failed && !line.isError;
          return (
            <div
              key={`line-${line.message}`}
              className={cn("flex items-start gap-2 py-0.5", line.isError && "text-destructive")}
            >
              <LineIcon isError={line.isError} isSpinning={isSpinning} />
              <span
                className={cn("text-left", !line.isError && !isSpinning && "text-muted-foreground")}
              >
                {line.message}
              </span>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>

      {failed && (
        <div className="flex flex-col items-center gap-2">
          <Button onClick={onReconnect}>Reconnect</Button>
          <Button variant="link" className="text-muted-foreground" onClick={onOpenClusterSettings}>
            Open Cluster Settings
          </Button>
        </div>
      )}
    </div>
  );
};
